package database_mgmt

import (
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ShardRequest struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	URL   string `json:"db_url"`
	Token string `json:"token"`
}

// ListInfrastructureShards returns the master shards stored in Core Master
func ListInfrastructureShards(c *fiber.Ctx) error {
	if database.Router == nil || database.Router.GetCoreMasterDB() == nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Core Master not connected"})
	}

	rows, err := database.Router.GetCoreMasterDB().Query("SELECT id, name, type, db_url, status, created_at FROM infrastructure_shards ORDER BY created_at DESC")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to query infrastructure"})
	}
	defer rows.Close()

	var shards []interface{}
	for rows.Next() {
		var id, name, sType, url, status string
		var createdAt int64
		if err := rows.Scan(&id, &name, &sType, &url, &status, &createdAt); err == nil {
			shards = append(shards, fiber.Map{
				"id":         id,
				"name":       name,
				"type":       sType,
				"db_url":     url,
				"status":     status,
				"created_at": createdAt,
			})
		}
	}

	return c.JSON(fiber.Map{"success": true, "shards": shards})
}

// AddInfrastructureShard registers a new master shard in the Core Master
func AddInfrastructureShard(c *fiber.Ctx) error {
	var req ShardRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	if req.URL == "" || req.Token == "" || req.Type == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "URL, Token and Type are required"})
	}

	// Encrypt Token using Master Key
	encrypted, err := security.Encrypt(req.Token, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Encryption failed"})
	}

	id := uuid.New().String()
	now := time.Now().Unix()

	_, err = database.Router.GetCoreMasterDB().Exec(
		"INSERT INTO infrastructure_shards (id, name, type, db_url, encrypted_token, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?)",
		id, req.Name, req.Type, req.URL, encrypted, now, now,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to save shard: " + err.Error()})
	}

	// 5. Hot-Attach Shard to Active Fleet
	go database.Router.RegisterShardDynamically(req.Type, req.URL, req.Token)

	return c.JSON(fiber.Map{"success": true, "message": "Infrastructure shard registered and attached successfully"})
}

// RemoveInfrastructureShard deletes a master shard from Core Master
func RemoveInfrastructureShard(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "ID is required"})
	}

	_, err := database.Router.GetCoreMasterDB().Exec("DELETE FROM infrastructure_shards WHERE id = ?", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to remove shard"})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Infrastructure shard removed"})
}
