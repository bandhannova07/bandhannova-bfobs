package storage_mgmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/gofiber/fiber/v2"
)

// CreateHuggingFaceRepo creates a new Dataset repository on Hugging Face
func CreateHuggingFaceRepo(c *fiber.Ctx) error {
	var req struct {
		Name    string `json:"name"`
		Private bool   `json:"private"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	token := config.AppConfig.HFToken
	if token == "" {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Hugging Face Token not configured"})
	}

	// HF API: POST /api/repos/create
	apiUrl := "https://huggingface.co/api/repos/create"
	payload := map[string]interface{}{
		"name":    req.Name,
		"type":    "dataset",
		"private": req.Private,
	}
	
	jsonPayload, _ := json.Marshal(payload)
	hReq, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create request"})
	}

	hReq.Header.Set("Authorization", "Bearer "+token)
	hReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(hReq)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to connect to Hugging Face"})
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": true,
			"message": "Failed to create HF Dataset",
			"details": string(body),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Cloud Storage Bucket created successfully",
		"data":    string(body),
	})
}

// UploadToHuggingFace sends a file to a Hugging Face repository
func UploadToHuggingFace(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "No file uploaded"})
	}

	// 1. Get HF Config from Environment
	token := config.AppConfig.HFToken
	repo := config.AppConfig.HFStorageRepo // e.g., "lordbandhan07/bandhannova-drive"
	
	if token == "" || repo == "" {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Hugging Face storage not configured"})
	}

	// 2. Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to open file"})
	}
	defer src.Close()

	fileBuffer := new(bytes.Buffer)
	if _, err := io.Copy(fileBuffer, src); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to read file"})
	}

	// 3. Prepare HF API Request
	productSlug := c.FormValue("product_slug", "general")
	fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
	hfPath := fmt.Sprintf("%s/uploads/%s", productSlug, fileName)
	
	// HF URL: https://huggingface.co/api/datasets/REPO/upload/main/PATH
	apiUrl := fmt.Sprintf("https://huggingface.co/api/datasets/%s/upload/main/%s", repo, hfPath)

	req, err := http.NewRequest("POST", apiUrl, fileBuffer)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create request"})
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/octet-stream")

	// 4. Send to Hugging Face
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to connect to Hugging Face"})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": true, 
			"message": "Hugging Face upload failed", 
			"details": string(body),
		})
	}

	// 5. Return the URL
	// Note: For private datasets, the raw URL requires a token. 
	// For public viewing, you might need to use a proxy or make the dataset public.
	rawUrl := fmt.Sprintf("https://huggingface.co/datasets/%s/resolve/main/%s", repo, hfPath)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "File uploaded to Hugging Face Cloud",
		"file_info": fiber.Map{
			"name": fileName,
			"path": hfPath,
			"url":  rawUrl,
			"size": file.Size,
		},
	})
}
