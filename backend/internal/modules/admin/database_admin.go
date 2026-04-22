package admin

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/database_mgmt"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

// ProvisionDatabase now handles manual registration of pre-created Turso DBs (Replaces automatic provisioning)
func ProvisionDatabase(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		Name      string `json:"name"`
		Category  string `json:"category"`
		URL       string `json:"db_url"`
		Token     string `json:"token"`
		ProductID string `json:"product_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	if req.URL == "" || req.Token == "" || req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Name, URL and Token are required for manual shard addition"})
	}

	// 1. Find which Global Manager Shard has this product (CRITICAL for Foreign Key)
	var targetGDB *sql.DB
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		var exists int
		err := gDB.QueryRow("SELECT 1 FROM managed_products WHERE id = ?", req.ProductID).Scan(&exists)
		if err == nil && exists == 1 {
			targetGDB = gDB
			break
		}
	}

	if targetGDB == nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Associated product not found on any global shard. Shard registration failed."})
	}

	// 2. Test Connection to the new shard
	testDB, err := database.ConnectTurso(req.URL, req.Token)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Failed to connect to the provided Turso URL: " + err.Error()})
	}
	defer testDB.Close()

	// 3. Encrypt Token
	encrypted, _ := security.Encrypt(req.Token, config.AppConfig.BandhanNovaMasterKey)

	// 4. Register in the correct Global Manager Shard
	id := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-")) + "-" + uuid.New().String()[:6]
	now := time.Now().Unix()

	_, err = targetGDB.Exec(
		"INSERT INTO managed_databases (id, slug, name, category, db_url, encrypted_token, product_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)",
		id, slug, req.Name, req.Category, req.URL, encrypted, req.ProductID, now, now,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to register shard: " + err.Error()})
	}

	// 5. Reload Managed DBs
	database_mgmt.ReloadManagedDatabases()

	LogAudit("ADD_PRODUCT_SHARD", req.Name, ip, fmt.Sprintf("Manually added shard: %s for product %s", req.Name, req.ProductID))

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Shard added successfully and linked to product",
		"slug":    slug,
	})
}

// BulkExecuteSQLHandler executes SQL on multiple shards for a specific product
func BulkExecuteSQLHandler(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		ProductID    string   `json:"product_id"`
		ShardSlugs   []string `json:"shard_slugs"`
		SQL          string   `json:"sql"`
		SaveToMaster bool     `json:"save_to_master"`
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

	// 2. Execute on each selected shard in parallel
	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, slug := range req.ShardSlugs {
		wg.Add(1)
		go func(s string) {
			defer wg.Done()
			res, err := database_mgmt.ExecuteSQL(s, req.SQL)

			mu.Lock()
			if err != nil {
				results[s] = fiber.Map{"success": false, "error": err.Error()}
			} else {
				results[s] = fiber.Map{"success": true, "result": res}
			}
			mu.Unlock()
		}(slug)
	}
	wg.Wait()

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
