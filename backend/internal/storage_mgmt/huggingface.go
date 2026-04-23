package storage_mgmt

import (
	"bytes"
	"database/sql"
	"encoding/base64"
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

type CommitFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"` // "base64"
}

type CommitPayload struct {
	Summary string       `json:"summary"`
	Files   []CommitFile `json:"files"`
}

// UploadToHuggingFace sends a file to a Hugging Face repository using the JSON Commit API (Standard)
func UploadToHuggingFace(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "No file uploaded"})
	}

	// 1. Get HF Config from Environment
	token := config.AppConfig.HFToken
	repo := config.AppConfig.HFStorageRepo

	if token == "" || repo == "" {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Hugging Face storage not configured"})
	}

	// 2. Read and Base64 encode the file
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to open file"})
	}
	defer src.Close()

	fileBytes, err := io.ReadAll(src)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to read file"})
	}
	encodedContent := base64.StdEncoding.EncodeToString(fileBytes)

	// 3. Prepare file path (Rename Logic: bandhannova-{14 digits}.ext)
	productSlug := c.FormValue("product_slug", "general")
	bucket := c.FormValue("bucket", "uploads")
	
	ext := "bin"
	if parts := strings.Split(file.Filename, "."); len(parts) > 1 {
		ext = parts[len(parts)-1]
	}
	
	// 14-digit code using timestamp: YYYYMMDDHHMMSS
	digitCode := time.Now().Format("20060102150405")
	fileName := fmt.Sprintf("bandhannova-%s.%s", digitCode, ext)
	hfPath := fmt.Sprintf("%s/%s/%s", productSlug, bucket, fileName)

	log.Printf("🚀 Pushing to HF Storage: %s | Repo: %s", hfPath, repo)

	payload := CommitPayload{
		Summary: fmt.Sprintf("Upload %s via BandhanNova BFOBS", fileName),
		Files: []CommitFile{
			{
				Path:     hfPath,
				Content:  encodedContent,
				Encoding: "base64",
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	// 4. Send to Hugging Face Commit Endpoint (Model Repository Method)
	apiUrl := fmt.Sprintf("https://huggingface.co/api/models/%s/commit/main", repo)

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("❌ Failed to create HF request: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create request"})
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ HF Connection Error: %v", err)
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to connect to Hugging Face"})
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("📥 HF API Response [%d]: %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error":   true,
			"message": "Hugging Face Commit failed",
			"details": string(respBody),
		})
	}

	// 5. Background Tracking (Optional metadata store)
	go func() {
		log.Printf("📝 Tracking file in Database: %s", fileName)
		var bucketID string
		var targetDB *sql.DB
		for _, gdb := range database.Router.GetAllGlobalManagerDBs() {
			err := gdb.QueryRow("SELECT b.id FROM storage_buckets b JOIN managed_products p ON b.product_id = p.id WHERE p.slug = ? AND b.slug = ?", productSlug, bucket).Scan(&bucketID)
			if err == nil {
				targetDB = gdb
				break
			}
		}

		if targetDB != nil {
			db := database.Router.GetManagedDBBySlug(productSlug)
			if db == nil { db = targetDB }
			_, _ = db.Exec(StorageAssetsSchema)
			assetID := uuid.New().String()
			contentType := file.Header.Get("Content-Type")
			_, _ = db.Exec("INSERT INTO storage_assets (id, bucket_id, name, path, size, content_type, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", assetID, bucketID, fileName, hfPath, file.Size, contentType, time.Now().Unix())
		}
	}()

	return c.JSON(fiber.Map{
		"success": true, 
		"message": "File uploaded successfully",
		"file": fiber.Map{
			"name": fileName,
			"path": hfPath,
			"url": fmt.Sprintf("/storage/view/%s/%s/%s", productSlug, bucket, fileName),
		},
	})
}

// DeleteFile removes a file from Hugging Face via Commit API
func DeleteFile(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	bucketSlug := c.Params("bucket_slug")
	fileName := c.Params("filename")

	token := config.AppConfig.HFToken
	repo := config.AppConfig.HFStorageRepo

	if token == "" || repo == "" {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Storage not configured"})
	}

	hfPath := fmt.Sprintf("%s/%s/%s", productSlug, bucketSlug, fileName)

	// HF Commit API structure for deletion
	payload := map[string]interface{}{
		"summary": fmt.Sprintf("Delete %s via BandhanNova BFOBS", fileName),
		"operations": []map[string]interface{}{
			{
				"operation": "delete",
				"path":      hfPath,
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	apiUrl := fmt.Sprintf("https://huggingface.co/api/models/%s/commit/main", repo)

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create request"})
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to connect to HF"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return c.Status(resp.StatusCode).JSON(fiber.Map{"error": true, "message": "HF Deletion failed", "details": string(body)})
	}

	// Optional: Remove from tracking DB if exists
	go func() {
		db := database.Router.GetManagedDBBySlug(productSlug)
		if db == nil { db = database.Router.GetGlobalManagerDB() }
		_, _ = db.Exec("DELETE FROM storage_assets WHERE name = ?", fileName)
	}()

	return c.JSON(fiber.Map{"success": true, "message": "File deleted successfully"})
}

// InitializeProductFolder creates a placeholder .keep file to "create" the product folder in HF
func InitializeProductFolder(slug string) {
	token := config.AppConfig.HFToken
	repo := config.AppConfig.HFStorageRepo
	if token == "" || repo == "" { return }

	hfPath := fmt.Sprintf("%s/.keep", slug)
	payload := CommitPayload{
		Summary: fmt.Sprintf("Initialize storage fleet for %s", slug),
		Files: []CommitFile{
			{
				Path:     hfPath,
				Content:  base64.StdEncoding.EncodeToString([]byte("Infrastructure active.")),
				Encoding: "base64",
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	apiUrl := fmt.Sprintf("https://huggingface.co/api/models/%s/commit/main", repo)

	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err == nil { defer resp.Body.Close() }
}

// ProxyViewFile allows viewing private files by proxying the request with the HF Token
func ProxyViewFile(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	bucket := c.Params("bucket_slug")
	fileName := c.Params("filename")

	token := config.AppConfig.HFToken
	repo := config.AppConfig.HFStorageRepo

	if token == "" || repo == "" {
		return c.Status(500).SendString("Storage not configured")
	}

	hfPath := fmt.Sprintf("%s/%s/%s", productSlug, bucket, fileName)
	apiUrl := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", repo, hfPath)

	log.Printf("🔍 Proxying Storage Request: %s", hfPath)

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return c.Status(500).SendString("Failed to create request")
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("❌ HF Proxy Error: %v", err)
		return c.Status(500).SendString("Failed to connect to storage")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("⚠️ HF Proxy returned %d for path: %s", resp.StatusCode, hfPath)
		return c.Status(resp.StatusCode).SendString("File not found or access denied")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(500).SendString("Failed to read storage response")
	}

	// 1. Get Content-Type from HF or fallback to detection
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		if strings.HasSuffix(strings.ToLower(fileName), ".png") { contentType = "image/png"
		} else if strings.HasSuffix(strings.ToLower(fileName), ".jpg") || strings.HasSuffix(strings.ToLower(fileName), ".jpeg") { contentType = "image/jpeg"
		} else if strings.HasSuffix(strings.ToLower(fileName), ".gif") { contentType = "image/gif"
		} else if strings.HasSuffix(strings.ToLower(fileName), ".webp") { contentType = "image/webp"
		} else if strings.HasSuffix(strings.ToLower(fileName), ".svg") { contentType = "image/svg+xml"
		} else if strings.HasSuffix(strings.ToLower(fileName), ".mp4") { contentType = "video/mp4"
		}
	}

	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(respBody)
}
