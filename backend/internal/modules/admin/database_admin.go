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

// UpdateDatabase handles modifying existing shard credentials
func UpdateDatabase(c *fiber.Ctx) error {
	id := c.Params("id")
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		Name      string `json:"name"`
		URL       string `json:"db_url"`
		Token     string `json:"token"`
		ProductID string `json:"product_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// Find the shard
	var targetGDB *sql.DB
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		var exists int
		err := gDB.QueryRow("SELECT 1 FROM managed_databases WHERE id = ?", id).Scan(&exists)
		if err == nil && exists == 1 {
			targetGDB = gDB
			break
		}
	}

	if targetGDB == nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Shard not found"})
	}

	now := time.Now().Unix()
	if req.Token != "" {
		encrypted, _ := security.Encrypt(req.Token, config.AppConfig.BandhanNovaMasterKey)
		_, err := targetGDB.Exec(
			"UPDATE managed_databases SET name = ?, db_url = ?, encrypted_token = ?, updated_at = ? WHERE id = ?",
			req.Name, req.URL, encrypted, now, id,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": true, "message": "Update failed"})
		}
	} else {
		_, err := targetGDB.Exec(
			"UPDATE managed_databases SET name = ?, db_url = ?, updated_at = ? WHERE id = ?",
			req.Name, req.URL, now, id,
		)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": true, "message": "Update failed"})
		}
	}

	database_mgmt.ReloadManagedDatabases()
	LogAudit("UPDATE_SHARD", req.Name, ip, fmt.Sprintf("Updated shard credentials for ID: %s", id))

	return c.JSON(fiber.Map{"success": true, "message": "Shard updated successfully"})
}

// BulkExecuteSQLHandler executes SQL on multiple shards for a specific product
func BulkExecuteSQLHandler(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		ProductID    string   `json:"product_id"`
		ProductSlug  string   `json:"product_slug"`
		ShardSlugs   []string `json:"shard_slugs"`
		SQL          string   `json:"sql"`
		SaveToMaster bool     `json:"save_to_master"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// 0. Resolve ShardSlugs if not provided but product_slug is
	if len(req.ShardSlugs) == 0 && (req.ProductSlug != "" || req.ProductID != "") {
		// Find the product and its shards
		var pID string = req.ProductID
		if pID == "" {
			// Find product ID by slug across global shards
			for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
				err := gDB.QueryRow("SELECT id FROM managed_products WHERE slug = ?", req.ProductSlug).Scan(&pID)
				if err == nil {
					break
				}
			}
		}

		if pID != "" {
			// Fetch all active shards for this product
			for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
				rows, err := gDB.Query("SELECT slug FROM managed_databases WHERE product_id = ? AND status = 'active'", pID)
				if err == nil {
					for rows.Next() {
						var sSlug string
						if err := rows.Scan(&sSlug); err == nil {
							req.ShardSlugs = append(req.ShardSlugs, sSlug)
						}
					}
					rows.Close()
				}
			}
		}
	}

	if len(req.ShardSlugs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "No active shards found for this product fleet"})
	}

	// 1. Save to Master Schema if requested
	if req.SaveToMaster && req.ProductID != "" {
		for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
			_, err := gDB.Exec(
				"UPDATE managed_products SET master_schema = ?, updated_at = ? WHERE id = ?",
				req.SQL, time.Now().Unix(), req.ProductID,
			)
			if err == nil {
				break
			}
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

	LogAudit("BULK_SQL_EXEC", req.ProductSlug, ip, fmt.Sprintf("Executed SQL on %d shards", len(req.ShardSlugs)))

	return c.JSON(fiber.Map{
		"success": true,
		"results": results,
		"shards_executed": len(req.ShardSlugs),
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

	log.Printf("[SQL Admin] Execution request for shard: %s | IP: %s", req.Shard, ip)

	result, err := database_mgmt.ExecuteSQL(req.Shard, req.SQL)
	if err != nil {
		if strings.Contains(err.Error(), "shard not found") {
			log.Printf("[SQL Admin] ❌ Shard not found in registry: %s", req.Shard)
			return c.Status(404).JSON(fiber.Map{"error": true, "message": "Target database shard (" + req.Shard + ") is offline or not registered."})
		}
		return c.Status(400).JSON(fiber.Map{"error": true, "message": err.Error()})
	}

	LogAudit("SQL_EXEC", req.Shard, ip, fmt.Sprintf("Executed: %s", strings.TrimSpace(req.SQL)))

	return c.JSON(fiber.Map{
		"success": true,
		"result":  result,
	})
}

// ExecuteCategorySQLHandler executes SQL on all shards belonging to a specific category (Admin only)
func ExecuteCategorySQLHandler(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		Category string `json:"category"`
		SQL      string `json:"sql"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// Find all shards in this category
	var shardSlugs []string
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		rows, err := gDB.Query("SELECT slug FROM managed_databases WHERE category = ? AND status = 'active'", req.Category)
		if err == nil {
			for rows.Next() {
				var slug string
				if err := rows.Scan(&slug); err == nil {
					shardSlugs = append(shardSlugs, slug)
				}
			}
			rows.Close()
		}
	}

	if len(shardSlugs) == 0 {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "No active shards found in this category"})
	}

	// Parallel Execution
	results := make(map[string]interface{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, slug := range shardSlugs {
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

	LogAudit("CATEGORY_SQL_EXEC", req.Category, ip, fmt.Sprintf("Executed SQL on %d shards in category %s", len(shardSlugs), req.Category))

	return c.JSON(fiber.Map{
		"success": true,
		"results": results,
		"shards_executed": len(shardSlugs),
	})
}

// ResetFleetHandler performs a factory reset on all shards for a specific product (Requires Infra ID)
func ResetFleetHandler(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req struct {
		ProductSlug string `json:"product_slug"`
		InfraID     string `json:"infra_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid request"})
	}

	// 1. Verify Infrastructure ID against the product
	var productID string
	var verified bool
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		// We use client_id (Infrastructure ID) for user-facing validation
		err := gDB.QueryRow("SELECT id FROM managed_products WHERE slug = ? AND client_id = ?", req.ProductSlug, req.InfraID).Scan(&productID)
		if err == nil {
			verified = true
			break
		}
	}

	if !verified {
		return c.Status(403).JSON(fiber.Map{"error": true, "message": "Unauthorized: Invalid Infrastructure ID for this product"})
	}

	// 2. Find all shards for this product
	var shardSlugs []string
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		rows, err := gDB.Query("SELECT slug FROM managed_databases WHERE product_id = ?", productID)
		if err == nil {
			for rows.Next() {
				var slug string
				if err := rows.Scan(&slug); err == nil {
					shardSlugs = append(shardSlugs, slug)
				}
			}
			rows.Close()
		}
	}

	// 3. Reset Each Shard (Drop all tables)
	// We'll get all tables and drop them one by one
	for _, slug := range shardSlugs {
		// Get tables
		res, err := database_mgmt.ExecuteSQL(slug, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
		if err == nil && res != nil {
			for _, row := range res.Rows {
				if tableName, ok := row["name"].(string); ok {
					database_mgmt.ExecuteSQL(slug, fmt.Sprintf("DROP TABLE IF EXISTS \"%s\"", tableName))
				}
			}
		}
	}

	LogAudit("FLEET_RESET", req.ProductSlug, ip, fmt.Sprintf("Factory reset performed on %d shards for product %s", len(shardSlugs), req.ProductSlug))

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("Fleet reset complete. %d shards were cleared.", len(shardSlugs)),
	})
}
