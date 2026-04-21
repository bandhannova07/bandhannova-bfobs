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

// UploadToHuggingFace sends a file to a Hugging Face repository using the new Commit API
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

	// 2. Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to open file"})
	}
	defer src.Close()

	// 3. Prepare Multipart Form for HF Commit API
	productSlug := c.FormValue("product_slug", "general")
	fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
	hfPath := fmt.Sprintf("%s/uploads/%s", productSlug, fileName)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add operations JSON
	// The API expects an array of operations. We use addFile to upload.
	ops := fmt.Sprintf(`[{"operation": "addFile", "pathInRepo": "%s"}]`, hfPath)
	_ = writer.WriteField("operations", ops)
	_ = writer.WriteField("commitMessage", fmt.Sprintf("Upload %s via BandhanNova API Hunter", fileName))

	// Add file content
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create form file"})
	}
	if _, err := io.Copy(part, src); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to copy file content"})
	}
	writer.Close()

	// 4. Send to Hugging Face Commit Endpoint
	// URL: https://huggingface.co/api/datasets/REPO/commit/main
	apiUrl := fmt.Sprintf("https://huggingface.co/api/datasets/%s/commit/main", repo)

	req, err := http.NewRequest("POST", apiUrl, body)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create request"})
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to connect to Hugging Face"})
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return c.Status(resp.StatusCode).JSON(fiber.Map{
			"error": true, 
			"message": "Hugging Face Commit failed", 
			"details": string(respBody),
		})
	}

	// 5. Return the URL
	rawUrl := fmt.Sprintf("https://huggingface.co/datasets/%s/resolve/main/%s", repo, hfPath)

	return c.JSON(fiber.Map{
		"success": true,
		"message": "File uploaded and committed to Hugging Face",
		"file_info": fiber.Map{
			"name": fileName,
			"path": hfPath,
			"url":  rawUrl,
			"size": file.Size,
		},
	})
}
