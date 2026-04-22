package database_mgmt

import (
	"fmt"
	"log"
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

	if database.Router == nil || database.Router.GetCoreMasterDB() == nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Core Master not connected. Check HF Secrets."})
	}

	// Self-healing: Ensure table exists
	database.InitInfrastructureSchema(database.Router.GetCoreMasterDB())

	// Encrypt Token using Master Key
	if config.AppConfig.BandhanNovaMasterKey == "" {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "BANDHANNOVA_MASTER_KEY is missing on server"})
	}

	encrypted, err := security.Encrypt(req.Token, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Encryption failed: " + err.Error()})
	}

	id := uuid.New().String()
	now := time.Now().Unix()

	_, err = database.Router.GetCoreMasterDB().Exec(
		"INSERT INTO infrastructure_shards (id, name, type, db_url, encrypted_token, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 'active', ?, ?)",
		id, req.Name, req.Type, req.URL, encrypted, now, now,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Database write failed: " + err.Error()})
	}

	// Hot-reload fleet registry
	go database.Router.RefreshFleet(config.AppConfig.BandhanNovaMasterKey)

	return c.JSON(fiber.Map{"success": true, "message": "Infrastructure shard registered successfully"})
}

// UpdateInfrastructureShard modifies an existing master shard
func UpdateInfrastructureShard(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "ID is required"})
	}

	var req ShardRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	if database.Router == nil || database.Router.GetCoreMasterDB() == nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Core Master not connected"})
	}

	now := time.Now().Unix()

	// If token is provided, encrypt it. Otherwise keep old one.
	if req.Token != "" {
		encrypted, err := security.Encrypt(req.Token, config.AppConfig.BandhanNovaMasterKey)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": true, "message": "Encryption failed"})
		}
		_, err = database.Router.GetCoreMasterDB().Exec(
			"UPDATE infrastructure_shards SET name = ?, type = ?, db_url = ?, encrypted_token = ?, updated_at = ? WHERE id = ?",
			req.Name, req.Type, req.URL, encrypted, now, id,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": true, "message": "Update failed"})
		}
	} else {
		_, err := database.Router.GetCoreMasterDB().Exec(
			"UPDATE infrastructure_shards SET name = ?, type = ?, db_url = ?, updated_at = ? WHERE id = ?",
			req.Name, req.Type, req.URL, now, id,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": true, "message": "Update failed"})
		}
	}

	// Hot-reload fleet registry
	go database.Router.RefreshFleet(config.AppConfig.BandhanNovaMasterKey)

	return c.JSON(fiber.Map{"success": true, "message": "Infrastructure shard updated"})
}

// QueryInfrastructureShard runs a SQL query on a specific shard from the fleet
func QueryInfrastructureShard(c *fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Query string `json:"query"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid query"})
	}

	// Find the shard in registry to get URL/Token
	var shard struct {
		URL   string `db:"db_url"`
		Token string `db:"encrypted_token"`
	}
	err := database.Router.GetCoreMasterDB().QueryRow(
		"SELECT db_url, encrypted_token FROM infrastructure_shards WHERE id = ?", id,
	).Scan(&shard.URL, &shard.Token)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Shard not found"})
	}

	decryptedToken, err := security.Decrypt(shard.Token, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to decrypt shard token"})
	}

	db, err := database.ConnectTurso(shard.URL, decryptedToken)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to connect to shard: " + err.Error()})
	}
	defer db.Close()

	rows, err := db.Query(req.Query)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Query failed: " + err.Error()})
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var result []map[string]interface{}
	for rows.Next() {
		columns := make([]interface{}, len(cols))
		columnPointers := make([]interface{}, len(cols))
		for i := range columns {
			columnPointers[i] = &columns[i]
		}
		rows.Scan(columnPointers...)
		
		rowMap := make(map[string]interface{})
		for i, colName := range cols {
			val := columns[i]
			if b, ok := val.([]byte); ok {
				rowMap[colName] = string(b)
			} else {
				rowMap[colName] = val
			}
		}
		result = append(result, rowMap)
	}

	return c.JSON(fiber.Map{"success": true, "data": result, "columns": cols})
}

// ClearInfrastructureShard wipes ALL database objects for a total reset
func ClearInfrastructureShard(c *fiber.Ctx) error {
	id := c.Params("id")

	var req struct {
		MasterKey string `json:"master_key"`
	}
	if err := c.BodyParser(&req); err != nil || req.MasterKey != config.AppConfig.BandhanNovaMasterKey {
		return c.Status(401).JSON(fiber.Map{"error": true, "message": "Security Verification Failed: Invalid Master Key"})
	}
	
	var shard struct {
		URL   string `db:"db_url"`
		Token string `db:"encrypted_token"`
		Type  string `db:"type"`
	}
	err := database.Router.GetCoreMasterDB().QueryRow(
		"SELECT db_url, encrypted_token, type FROM infrastructure_shards WHERE id = ?", id,
	).Scan(&shard.URL, &shard.Token, &shard.Type)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Shard not found"})
	}

	// Protect critical infrastructure
	if shard.Type == "auth" || shard.Type == "analytics" || shard.Type == "global_manager" {
		return c.Status(403).JSON(fiber.Map{
			"error": true,
			"message": "Critical Infrastructure Protection: Cannot wipe " + shard.Type + " nodes. This action is restricted to system-level maintenance.",
		})
	}

	decryptedToken, err := security.Decrypt(shard.Token, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Security failure"})
	}

	db, err := database.ConnectTurso(shard.URL, decryptedToken)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Connection failure"})
	}
	defer db.Close()

	// Disable foreign keys for aggressive drop
	db.Exec("PRAGMA foreign_keys = OFF")

	// Drop all objects (Tables, Indexes, Views, Triggers)
	rows, _ := db.Query("SELECT name, type FROM sqlite_master WHERE name NOT LIKE 'sqlite_%'")
	for rows.Next() {
		var name, objType string
		rows.Scan(&name, &objType)
		db.Exec(fmt.Sprintf("DROP %s IF EXISTS %s", objType, name))
	}
	rows.Close()
	
	db.Exec("PRAGMA foreign_keys = ON")

	return c.JSON(fiber.Map{"success": true, "message": "Shard totally wiped. Re-initialization required on close."})
}

// InitializeInfrastructureShard re-runs migrations for a shard based on its role
func InitializeInfrastructureShard(c *fiber.Ctx) error {
	id := c.Params("id")
	
	var shard struct {
		URL   string `db:"db_url"`
		Token string `db:"encrypted_token"`
		Type  string `db:"type"`
	}
	err := database.Router.GetCoreMasterDB().QueryRow(
		"SELECT db_url, encrypted_token, type FROM infrastructure_shards WHERE id = ?", id,
	).Scan(&shard.URL, &shard.Token, &shard.Type)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Shard not found"})
	}

	decryptedToken, err := security.Decrypt(shard.Token, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Security failure"})
	}

	db, err := database.ConnectTurso(shard.URL, decryptedToken)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Connection failure"})
	}
	defer db.Close()

	// Re-run migrations based on type
	log.Printf("🛠️  Initializing shard %s (Type: %s)", id, shard.Type)
	var migrationErr error
	switch shard.Type {
	case "global_manager":
		migrationErr = database.InitGlobalManagerSchema(db)
	case "auth":
		migrationErr = database.InitAuthSchema(db)
	case "analytics":
		migrationErr = database.InitAnalyticsSchema(db)
	case "user":
		migrationErr = database.InitUserSchema(db)
	default:
		log.Printf("⚠️  Unknown shard type: %s", shard.Type)
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Unknown shard type: " + shard.Type})
	}

	if migrationErr != nil {
		log.Printf("❌ Migration failed for shard %s: %v", id, migrationErr)
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Migration failed: " + migrationErr.Error()})
	}

	log.Printf("✅ Shard %s re-initialized successfully", id)
	return c.JSON(fiber.Map{"success": true, "message": "Shard re-initialized with " + shard.Type + " schema"})
}

// RemoveInfrastructureShard deletes a master shard from Core Master
func RemoveInfrastructureShard(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "ID is required"})
	}

	var req struct {
		MasterKey string `json:"master_key"`
	}
	if err := c.BodyParser(&req); err != nil || req.MasterKey != config.AppConfig.BandhanNovaMasterKey {
		return c.Status(401).JSON(fiber.Map{"error": true, "message": "Security Verification Failed: Invalid Master Key"})
	}

	// Protect critical infrastructure
	var sType string
	err := database.Router.GetCoreMasterDB().QueryRow("SELECT type FROM infrastructure_shards WHERE id = ?", id).Scan(&sType)
	if err == nil {
		if sType == "auth" || sType == "analytics" || sType == "global_manager" {
			return c.Status(403).JSON(fiber.Map{
				"error": true,
				"message": "Critical Infrastructure Protection: Cannot remove " + sType + " nodes via this endpoint.",
			})
		}
	}

	_, err = database.Router.GetCoreMasterDB().Exec("DELETE FROM infrastructure_shards WHERE id = ?", id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to remove shard"})
	}

	// Hot-reload fleet registry
	go database.Router.RefreshFleet(config.AppConfig.BandhanNovaMasterKey)

	return c.JSON(fiber.Map{"success": true, "message": "Infrastructure shard removed"})
}
