package storage_mgmt

import (
	"bytes"
	"encoding/base64"
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

type CommitOperation struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Content   string `json:"content"`
}

type CommitPayload struct {
	Summary    string            `json:"summary"`
	Operations []CommitOperation `json:"operations"`
}

// UploadToHuggingFace sends a file to a Hugging Face repository using the JSON Commit API
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

	// 3. Prepare JSON Payload for HF Commit API
	productSlug := c.FormValue("product_slug", "general")
	fileName := fmt.Sprintf("%d-%s", time.Now().Unix(), file.Filename)
	hfPath := fmt.Sprintf("%s/uploads/%s", productSlug, fileName)

	payload := CommitPayload{
		Summary: fmt.Sprintf("Upload %s via BandhanNova API Hunter", fileName),
		Operations: []CommitOperation{
			{
				Operation: "add",
				Path:      hfPath,
				Content:   encodedContent,
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)

	// 4. Send to Hugging Face Commit Endpoint
	apiUrl := fmt.Sprintf("https://huggingface.co/api/datasets/%s/commit/main", repo)

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create request"})
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

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
	})
}

// InitializeProductFolder creates a placeholder .keep file to "create" the product folder in HF
func InitializeProductFolder(slug string) {
	token := config.AppConfig.HFToken
	repo := config.AppConfig.HFStorageRepo
	if token == "" || repo == "" {
		return
	}

	hfPath := fmt.Sprintf("%s/.keep", slug)
	payload := CommitPayload{
		Summary: fmt.Sprintf("Initialize storage fleet for %s", slug),
		Operations: []CommitOperation{
			{
				Operation: "add",
				Path:      hfPath,
				Content:   base64.StdEncoding.EncodeToString([]byte("Infrastructure active.")),
			},
		},
	}

	jsonPayload, _ := json.Marshal(payload)
	apiUrl := fmt.Sprintf("https://huggingface.co/api/datasets/%s/commit/main", repo)

	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}
}
