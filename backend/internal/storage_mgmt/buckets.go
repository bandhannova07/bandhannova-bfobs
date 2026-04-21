package storage_mgmt

import (
	"fmt"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
    "strings"
)

type Bucket struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
	CreatedAt   int64  `json:"created_at"`
}

// ListBuckets returns all buckets for a specific product
func ListBuckets(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	
	// First get the product ID
	var productID string
	err := database.Router.GetGlobalManagerDB().QueryRow("SELECT id FROM managed_products WHERE slug = ?", productSlug).Scan(&productID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Product not found"})
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(
		"SELECT id, name, slug, description, is_public, created_at FROM storage_buckets WHERE product_id = ?",
		productID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to fetch buckets"})
	}
	defer rows.Close()

	var buckets []Bucket
	for rows.Next() {
		var b Bucket
		var isPublic int
		if err := rows.Scan(&b.ID, &b.Name, &b.Slug, &b.Description, &isPublic, &b.CreatedAt); err != nil {
			continue
		}
		b.IsPublic = isPublic == 1
		b.ProductID = productID
		buckets = append(buckets, b)
	}

	return c.JSON(fiber.Map{"success": true, "buckets": buckets})
}

// CreateBucket registers a new bucket and initializes it on Hugging Face
func CreateBucket(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Bucket name is required"})
	}

	// 1. Get Product ID
	var productID string
	err := database.Router.GetGlobalManagerDB().QueryRow("SELECT id FROM managed_products WHERE slug = ?", productSlug).Scan(&productID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Product not found"})
	}

	id := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	now := time.Now().Unix()
	isPublicInt := 0
	if req.IsPublic {
		isPublicInt = 1
	}

	// 2. Save to Database
	_, err = database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO storage_buckets (id, product_id, name, slug, description, is_public, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
		id, productID, req.Name, slug, req.Description, isPublicInt, now,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": true, 
			"message": "Failed to create bucket in DB",
			"details": err.Error(),
		})
	}

	// 3. Initialize on Hugging Face (Create .keep file)
	// We use /{product-slug}/{bucket-slug}/.keep
	hfPath := fmt.Sprintf("%s/%s", productSlug, slug)
	go InitializeProductFolder(hfPath) // Reusing existing logic but with bucket path

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Bucket created successfully",
		"bucket": fiber.Map{
			"id": id,
			"name": req.Name,
			"slug": slug,
			"upload_url": fmt.Sprintf("/v1/storage/upload?product_slug=%s&bucket=%s", productSlug, slug),
			"view_url": fmt.Sprintf("/storage/view/%s/%s/{filename}", productSlug, slug),
		},
	})
}

// DeleteBucket removes a bucket. Requires Master Key validation.
func DeleteBucket(c *fiber.Ctx) error {
	bucketID := c.Params("id")
	masterKey := c.Get("X-BandhanNova-Master-Key")
	confirmText := c.Query("confirm")

	if masterKey != config.AppConfig.BandhanNovaMasterKey {
		return c.Status(401).JSON(fiber.Map{"error": true, "message": "Unauthorized: Invalid Master Key"})
	}

	// Fetch bucket details first for confirmation
	var bucketName, productID string
	err := database.Router.GetGlobalManagerDB().QueryRow("SELECT name, product_id FROM storage_buckets WHERE id = ?", bucketID).Scan(&bucketName, &productID)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Bucket not found"})
	}

	expectedConfirm := fmt.Sprintf("I am Bandhan, I want to delete this bucket named %s.", bucketName)
	if confirmText != expectedConfirm {
		return c.Status(400).JSON(fiber.Map{
			"error": true, 
			"message": "Confirmation text mismatch",
			"expected": expectedConfirm,
		})
	}

	// Delete from DB
	_, err = database.Router.GetGlobalManagerDB().Exec("DELETE FROM storage_buckets WHERE id = ?", bucketID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to delete bucket from DB"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Bucket decommissioned successfully"})
}

// ListBucketFiles fetches the file list from our Database (Supabase Style)
func ListBucketFiles(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	bucketSlug := c.Params("bucket_slug")

	// Resolve bucket ID
	var bucketID string
	err := database.Router.GetGlobalManagerDB().QueryRow(
		"SELECT b.id FROM storage_buckets b JOIN managed_products p ON b.product_id = p.id WHERE p.slug = ? AND b.slug = ?",
		productSlug, bucketSlug,
	).Scan(&bucketID)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Bucket not found"})
	}

	rows, err := database.Router.GetGlobalManagerDB().Query(
		"SELECT id, name, path, size, content_type, created_at FROM storage_assets WHERE bucket_id = ? ORDER BY created_at DESC",
		bucketID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to fetch assets"})
	}
	defer rows.Close()

	var files []interface{}
	for rows.Next() {
		var id, name, path, contentType string
		var size int64
		var createdAt int64
		if err := rows.Scan(&id, &name, &path, &size, &contentType, &createdAt); err != nil {
			continue
		}

		// Map to a format the frontend understands (mimicking HF Tree API structure for compatibility)
		files = append(files, fiber.Map{
			"id":   id,
			"path": path,
			"name": name,
			"type": "file",
			"size": size,
			"lastCommit": fiber.Map{
				"date": time.Unix(createdAt, 0).Format(time.RFC3339),
			},
		})
	}

	return c.JSON(fiber.Map{"success": true, "files": files})
}
