package storage_mgmt

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// StorageAssetsSchema ensures we have a table to track assets if needed
const StorageAssetsSchema = `
CREATE TABLE IF NOT EXISTS storage_assets (
    id VARCHAR(255) PRIMARY KEY,
    bucket_id VARCHAR(255),
    name TEXT,
    path TEXT,
    size BIGINT,
    content_type VARCHAR(100),
    created_at BIGINT
);
`

// CreateBucket creates a new storage bucket for a product
func CreateBucket(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	var req struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// 1. Resolve Product ID
	var productID string
	err := database.Router.GetGlobalManagerDB().QueryRow("SELECT id FROM managed_products WHERE slug = ?", productSlug).Scan(&productID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Product not found"})
	}

	// 2. Insert into Global Manager (storage_buckets table)
	bucketID := uuid.New().String()
	_, err = database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO storage_buckets (id, product_id, name, slug, created_at) VALUES (?, ?, ?, ?, ?)",
		bucketID, productID, req.Name, req.Slug, time.Now().Unix(),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create bucket"})
	}

	// 3. Optional: Initialize folder in HF
	go InitializeProductFolder(productSlug + "/" + req.Slug)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Bucket created successfully",
		"bucket": fiber.Map{
			"id":   bucketID,
			"name": req.Name,
			"slug": req.Slug,
		},
	})
}

// ListBuckets returns all buckets for a product
func ListBuckets(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")

	var buckets []fiber.Map

	// We need to query ALL global manager shards because products are distributed
	for _, db := range database.Router.GetAllGlobalManagerDBs() {
		rows, err := db.Query(`
			SELECT b.id, b.name, b.slug, b.created_at 
			FROM storage_buckets b 
			JOIN managed_products p ON b.product_id = p.id 
			WHERE p.slug = ?`, productSlug)

		if err != nil {
			continue
		}
		defer rows.Close()

		for rows.Next() {
			var id, name, slug string
			var createdAt int64
			rows.Scan(&id, &name, &slug, &createdAt)
			buckets = append(buckets, fiber.Map{
				"id":         id,
				"name":       name,
				"slug":       slug,
				"created_at": createdAt,
			})
		}
	}

	return c.JSON(fiber.Map{"success": true, "buckets": buckets})
}

// DeleteBucket removes a bucket. Requires Master Key validation.
func DeleteBucket(c *fiber.Ctx) error {
	bucketID := c.Params("id")
	masterKey := c.Get("X-BandhanNova-Master-Key")
	confirmText := c.Query("confirm")

	if masterKey != config.AppConfig.BandhanNovaMasterKey {
		return c.Status(401).JSON(fiber.Map{"error": true, "message": "Unauthorized: Invalid Master Key"})
	}

	// 1. Resolve bucket ID and the DB it lives on by searching all shards
	var bucketName, productID string
	var targetDB *sql.DB

	for _, db := range database.Router.GetAllGlobalManagerDBs() {
		err := db.QueryRow("SELECT name, product_id FROM storage_buckets WHERE id = ?", bucketID).Scan(&bucketName, &productID)
		if err == nil {
			targetDB = db
			break
		}
	}

	if targetDB == nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Bucket not found"})
	}

	expectedConfirm := fmt.Sprintf("I am Bandhan, I want to delete this bucket named %s.", bucketName)
	if confirmText != expectedConfirm {
		return c.Status(400).JSON(fiber.Map{
			"error":    true,
			"message":  "Confirmation text mismatch",
			"expected": expectedConfirm,
		})
	}

	// 2. Delete from DB (on the correct shard)
	_, err := targetDB.Exec("DELETE FROM storage_buckets WHERE id = ?", bucketID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to delete bucket from DB"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Bucket decommissioned successfully"})
}

// ListBucketFiles queries Hugging Face to get a list of files in a bucket
func ListBucketFiles(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	bucketSlug := c.Params("bucket_slug")

	token := config.AppConfig.HFToken
	repo := config.AppConfig.HFStorageRepo

	if token == "" || repo == "" {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Storage not configured"})
	}

	// Query HF Tree API directly — this is the source of truth (Model Repo)
	hfPath := fmt.Sprintf("%s/%s", productSlug, bucketSlug)
	apiUrl := fmt.Sprintf("https://huggingface.co/api/models/%s/tree/main/%s", repo, hfPath)

	log.Printf("📂 Listing HF bucket: %s", hfPath)

	req, _ := http.NewRequest("GET", apiUrl, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to connect to Hugging Face"})
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Path doesn't exist yet, return empty list instead of error
		return c.JSON(fiber.Map{"success": true, "files": []fiber.Map{}})
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("⚠️ HF Tree API returned %d for %s: %s", resp.StatusCode, hfPath, string(body))
		return c.Status(resp.StatusCode).JSON(fiber.Map{"error": true, "message": "Hugging Face API error"})
	}

	// Parse HF Tree Response
	var hfFiles []struct {
		Type         string `json:"type"`
		Path         string `json:"path"`
		Size         int64  `json:"size"`
		LastModified string `json:"lastModified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&hfFiles); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to parse HF response"})
	}

	var files []fiber.Map
	for _, f := range hfFiles {
		if f.Type == "file" && !strings.HasSuffix(f.Path, ".keep") {
			// Extract filename from path
			pathParts := strings.Split(f.Path, "/")
			fileName := pathParts[len(pathParts)-1]

			files = append(files, fiber.Map{
				"name":          fileName,
				"path":          f.Path,
				"size":          f.Size,
				"url":           fmt.Sprintf("/storage/view/%s/%s/%s", productSlug, bucketSlug, fileName),
				"type":          f.Type,
				"last_modified": f.LastModified,
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"count":   len(files),
		"files":   files,
	})
}
