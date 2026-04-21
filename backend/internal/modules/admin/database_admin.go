package admin

import (
	"database/sql"
	"fmt"
	"log"
	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/database_mgmt"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"strings"
	"time"
)

// ListAllDatabases returns metrics for all core and managed shards
func ListAllDatabases(c *fiber.Ctx) error {
	var shards []ShardStatus
	
	// Core Shards
	shards = append(shards, getShardMetrics(database.Router.GetAuthDB(), "Auth Shard", "core", []string{"users", "sessions"}))
	shards = append(shards, getShardMetrics(database.Router.GetAnalyticsDB(), "Analytics Shard", "core", []string{"request_logs"}))
	shards = append(shards, getShardMetrics(database.Router.GetGlobalManagerDB(), "Global Manager", "core", []string{"api_cards", "managed_api_keys"}))

	// Managed Shards
	managed := database.Router.GetAllManagedDBs()
	for _, m := range managed {
		shards = append(shards, getShardMetrics(m.DB, m.Name, m.Category, []string{}))
	}

	return c.JSON(fiber.Map{
		"success": true,
		"shards":  shards,
	})
}

// ProvisionDatabase creates a new Turso DB dynamically
func ProvisionDatabase(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		Name      string `json:"name"`
		Category  string `json:"category"`
		ProductID string `json:"product_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// 1. Create on Turso
	dbInfo, err := database_mgmt.CreateTursoDatabase(req.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": fmt.Sprintf("Turso Creation Failed: %v", err)})
	}

	// 2. Create Token
	token, err := database_mgmt.CreateTursoToken(req.Name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Token Generation Failed"})
	}

	dbURL := fmt.Sprintf("libsql://%s", dbInfo.Hostname)

	// 3. Encrypt Token
	encrypted, _ := security.Encrypt(token, config.AppConfig.BandhanNovaMasterKey)

	// 4. Register in Global Manager
	id := uuid.New().String()
	_, err = database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO managed_databases (id, slug, name, category, db_url, encrypted_token, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		id, req.Name, req.Name, req.Category, dbURL, encrypted, time.Now().Unix(), time.Now().Unix(),
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to register database locally"})
	}

	// 5. Reload Managed DBs
	database_mgmt.ReloadManagedDatabases()

	// 6. AUTO-SYNC: Run Master Schema if exists for the product
	if req.ProductID != "" {
		var schema sql.NullString
		database.Router.GetGlobalManagerDB().QueryRow("SELECT master_schema FROM managed_products WHERE id = ?", req.ProductID).Scan(&schema)
		if schema.Valid && schema.String != "" {
			database_mgmt.ExecuteSQL(req.Name, schema.String)
		}
	}

	LogAudit("PROVISION_DB", req.Name, ip, fmt.Sprintf("Provisioned new shard: %s (%s)", req.Name, req.Category))

	return c.JSON(fiber.Map{
		"success": true,
		"db": fiber.Map{
			"url":   dbURL,
			"token": token,
		},
	})
}

// BulkExecuteSQLHandler executes SQL on multiple shards for a specific product
func BulkExecuteSQLHandler(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		ProductID string   `json:"product_id"`
		ShardSlugs []string `json:"shard_slugs"`
		SQL        string   `json:"sql"`
		SaveToMaster bool    `json:"save_to_master"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// 1. Save to Master Schema if requested
	if req.SaveToMaster && req.ProductID != "" {
		_, err := database.Router.GetGlobalManagerDB().Exec(
			"UPDATE managed_products SET master_schema = ?, updated_at = ? WHERE id = ?",
			req.SQL, time.Now().Unix(), req.ProductID,
		)
		if err != nil {
			log.Printf("Failed to update master schema: %v", err)
		}
	}

	// 2. Execute on each selected shard
	results := make(map[string]interface{})
	for _, slug := range req.ShardSlugs {
		res, err := database_mgmt.ExecuteSQL(slug, req.SQL)
		if err != nil {
			results[slug] = fiber.Map{"success": false, "error": err.Error()}
		} else {
			results[slug] = fiber.Map{"success": true, "result": res}
		}
	}

	LogAudit("BULK_SQL_EXEC", req.ProductID, ip, fmt.Sprintf("Executed SQL on %d shards", len(req.ShardSlugs)))

	return c.JSON(fiber.Map{
		"success": true,
		"results": results,
	})
}

// ExecuteSQLHandler handles raw SQL execution requests
func ExecuteSQLHandler(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		Shard string `json:"shard"`
		SQL   string `json:"sql"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// Safety check: Don't allow destructive queries on core shards via UI unless specifically allowed
	// For now, allow everything for the "Master Admin"
	
	result, err := database_mgmt.ExecuteSQL(req.Shard, req.SQL)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": err.Error()})
	}

	LogAudit("SQL_EXEC", req.Shard, ip, fmt.Sprintf("Executed: %s", strings.TrimSpace(req.SQL)))

	return c.JSON(fiber.Map{
		"success": true,
		"result":  result,
	})
}
