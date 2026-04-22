package database_mgmt

import (
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/gofiber/fiber/v2"
)

// ProxyDatabaseRequest represents a query coming from an external application
type ProxyDatabaseRequest struct {
	SQL    string        `json:"sql"`
	Params []interface{} `json:"params"`
}

// DatabaseProxyHandler routes virtual database requests to physical Turso shards
// Authentication: Uses product access_token (NOT master key)
func DatabaseProxyHandler(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	accessToken := c.Get("X-BandhanNova-Token")

	if productSlug == "" || accessToken == "" {
		return c.Status(401).JSON(fiber.Map{
			"error":   true,
			"message": "Authentication required. Provide X-BandhanNova-Token header with your product access_token.",
		})
	}

	// 1. Fleet-wide Product Resolution: Search ALL Global Manager shards
	var productID string
	found := false
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		var pid, storedToken string
		err := gDB.QueryRow(
			"SELECT id, access_token FROM managed_products WHERE slug = ? AND status = 'active'",
			productSlug,
		).Scan(&pid, &storedToken)
		if err == nil {
			// Verify access token
			if storedToken != accessToken {
				return c.Status(403).JSON(fiber.Map{
					"error":   true,
					"message": "Invalid access token for this product.",
				})
			}
			productID = pid
			found = true
			break
		}
	}

	if !found {
		return c.Status(404).JSON(fiber.Map{
			"error":   true,
			"message": "Product not found in the fleet registry.",
		})
	}

	// 2. Load Shards for this product (fleet-wide search)
	var shardSlugs []string
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		rows, err := gDB.Query(
			"SELECT slug FROM managed_databases WHERE product_id = ? AND status = 'active'",
			productID,
		)
		if err != nil {
			continue
		}
		for rows.Next() {
			var s string
			rows.Scan(&s)
			shardSlugs = append(shardSlugs, s)
		}
		rows.Close()
	}

	if len(shardSlugs) == 0 {
		return c.Status(503).JSON(fiber.Map{
			"error":   true,
			"message": "No active database shards available for this product.",
		})
	}

	// 3. Parse Query
	var req ProxyDatabaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid SQL payload. Send {\"sql\": \"...\", \"params\": [...]}"})
	}

	if req.SQL == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "SQL query is required."})
	}

	// 4. Execute with Failover Support (try each shard until one succeeds)
	var result *QueryResult
	var execErr error
	success := false

	for _, targetShard := range shardSlugs {
		result, execErr = ExecuteSQL(targetShard, req.SQL)
		if execErr == nil {
			success = true
			break
		}
	}

	if !success {
		errMsg := "All assigned shards are unreachable."
		if execErr != nil {
			errMsg = execErr.Error()
		}
		return c.Status(500).JSON(fiber.Map{
			"error":   true,
			"message": "Service Unstable: " + errMsg,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"result":  result,
	})
}
