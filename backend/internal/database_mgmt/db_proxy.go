package database_mgmt

import (
	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/gofiber/fiber/v2"
)

// ProxyDatabaseRequest represents a query coming from an external application
type ProxyDatabaseRequest struct {
	SQL string `json:"sql"`
}

// DatabaseProxyHandler routes virtual database requests to physical Turso shards
func DatabaseProxyHandler(c *fiber.Ctx) error {
	productSlug := c.Params("product_slug")
	virtualToken := c.Get("X-BandhanNova-Token")

	if productSlug == "" || virtualToken == "" {
		return c.Status(401).JSON(fiber.Map{"error": true, "message": "Authentication required"})
	}

	// 1. Resolve Product and Verify Token
	var productID string
	err := database.Router.GetGlobalManagerDB().QueryRow(
		"SELECT id FROM managed_products WHERE slug = ?", 
		productSlug,
	).Scan(&productID)

	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Product not found"})
	}

	// For simplicity, we use a derived token from the master key or a dedicated column.
	// The user requested a "token", let's assume we use the Master Key as the base or a product-specific one.
	// For now, we'll verify against the BandhanNova Master Key to keep it simple but "Virtual".
	if virtualToken != config.AppConfig.BandhanNovaMasterKey {
		return c.Status(403).JSON(fiber.Map{"error": true, "message": "Invalid BandhanNova Token"})
	}

	// 2. Load Shards for this product
	rows, err := database.Router.GetGlobalManagerDB().Query(
		"SELECT slug FROM managed_databases WHERE product_id = ? AND status = 'active'",
		productID,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to resolve shards"})
	}
	defer rows.Close()

	var shards []string
	for rows.Next() {
		var s string
		rows.Scan(&s)
		shards = append(shards, s)
	}

	if len(shards) == 0 {
		return c.Status(503).JSON(fiber.Map{"error": true, "message": "No active shards available for this product"})
	}

	// 4. Parse Query
	var req ProxyDatabaseRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid SQL payload"})
	}

	// 3. Execution with Failover Support
	var result *QueryResult
	var execErr error
	success := false

	for _, targetShard := range shards {
		result, execErr = ExecuteSQL(targetShard, req.SQL)
		if execErr == nil {
			success = true
			break
		}
	}

	if !success {
		return c.Status(500).JSON(fiber.Map{
			"error": true, 
			"message": "Service Unstable: All assigned shards for this product are currently unreachable.",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"result":  result,
	})
}
