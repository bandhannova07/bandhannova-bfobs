package database_mgmt

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/modules/admin"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type DatabaseResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	URL       string `json:"db_url"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	IsCore    bool   `json:"is_core"` // True if loaded from .env
}

// ReloadManagedDatabases hot-swaps active DBs from the global managed_databases table
func ReloadManagedDatabases() error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return fmt.Errorf("global DB not connected")
	}

	rows, err := database.Router.GetGlobalManagerDB().Query("SELECT id, slug, name, category, db_url, encrypted_token FROM managed_databases WHERE status = 'active'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var mDBs []database.ManagedDB
	for rows.Next() {
		var id, slug, name, category, dbURL, encrypted string
		if err := rows.Scan(&id, &slug, &name, &category, &dbURL, &encrypted); err != nil {
			continue
		}

		token, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
		if err != nil {
			log.Printf("Failed to decrypt DB token for %s", slug)
			continue
		}

		// Connect to the DB
		db, err := database.ConnectTurso(dbURL, token)
		if err != nil {
			log.Printf("Failed to connect to managed DB %s: %v", slug, err)
			continue
		}

		mDBs = append(mDBs, database.ManagedDB{
			Slug:     slug,
			Name:     name,
			Category: category,
			DB:       db,
		})
	}

	database.Router.ReloadDynamicDBs(mDBs)
	return nil
}

// ReloadManagedAPIKeys hot-reloads API keys from the global managed_api_keys table
func ReloadManagedAPIKeys() error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return fmt.Errorf("global DB not connected")
	}

	rows, err := database.Router.GetGlobalManagerDB().Query("SELECT provider, encrypted_value FROM managed_api_keys WHERE status = 'active'")
	if err != nil {
		return err
	}
	defer rows.Close()

	providerKeys := make(map[string][]string)
	for rows.Next() {
		var provider, encrypted string
		if err := rows.Scan(&provider, &encrypted); err != nil {
			continue
		}

		key, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
		if err != nil {
			log.Printf("Failed to decrypt API key for provider: %s", provider)
			continue
		}

		providerKeys[provider] = append(providerKeys[provider], key)
	}

	for provider, keys := range providerKeys {
		config.UpdateKeys(provider, keys)
	}

	log.Printf("🛡️  Managed API keys reloaded from database")
	return nil
}

// ListDatabases returns all active databases (core + managed)
func ListDatabases(c *fiber.Ctx) error {
	var resp []DatabaseResponse

	// Add Core DBs
	if config.AppConfig.TursoAuthURL != "" {
		resp = append(resp, DatabaseResponse{ID: "core-auth", Slug: "core-auth", Name: "Core Auth", Category: "auth", URL: config.AppConfig.TursoAuthURL, Status: "active", IsCore: true})
		resp = append(resp, DatabaseResponse{ID: "core-analytics", Slug: "core-analytics", Name: "Core Analytics", Category: "analytics", URL: config.AppConfig.TursoAnalyticsURL, Status: "active", IsCore: true})
		resp = append(resp, DatabaseResponse{ID: "core-global", Slug: "core-global", Name: "Core Global", Category: "global", URL: config.AppConfig.TursoGlobalURL, Status: "active", IsCore: true})
		for i, u := range config.AppConfig.TursoUserShardURLs {
			slug := fmt.Sprintf("core-user-%d", i)
			resp = append(resp, DatabaseResponse{ID: slug, Slug: slug, Name: fmt.Sprintf("Core User Shard %d", i), Category: "user", URL: u, Status: "active", IsCore: true})
		}
	}

	// Add Managed DBs
	if database.Router != nil && database.Router.GetGlobalManagerDB() != nil {
		rows, err := database.Router.GetGlobalManagerDB().Query("SELECT id, slug, name, category, db_url, status, created_at, updated_at FROM managed_databases ORDER BY created_at DESC")
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var d DatabaseResponse
				if err := rows.Scan(&d.ID, &d.Slug, &d.Name, &d.Category, &d.URL, &d.Status, &d.CreatedAt, &d.UpdatedAt); err == nil {
					d.IsCore = false
					resp = append(resp, d)
				}
			}
		}
	}

	return c.JSON(fiber.Map{"success": true, "databases": resp})
}

type AddDBRequest struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	URL      string `json:"db_url"`
	Token    string `json:"token"`
}

// AddDatabase adds a new dynamic database
func AddDatabase(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req AddDBRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	if req.Name == "" || req.URL == "" || req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Name, URL, and Token required"})
	}

	validCats := map[string]bool{"auth": true, "analytics": true, "global": true, "user": true}
	if !validCats[req.Category] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Invalid category"})
	}

	// 1. Test Connection
	testDB, err := database.ConnectTurso(req.URL, req.Token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Failed to connect to Turso database. Verify URL and Token."})
	}
	defer testDB.Close()

	// 2. Encrypt Token
	encrypted, err := security.Encrypt(req.Token, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Encryption failed"})
	}

	// 3. Generate Slug & Save
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-")) + "-" + uuid.New().String()[:6]
	id := uuid.New().String()
	now := time.Now().Unix()

	_, err = database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO managed_databases (id, slug, name, category, db_url, encrypted_token, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)",
		id, slug, req.Name, req.Category, req.URL, encrypted, now, now,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Failed to save database config"})
	}

	admin.LogAudit("ADD_DATABASE", req.Category, ip, fmt.Sprintf("Added DB: %s (%s)", req.Name, req.URL))

	// Hot Reload
	go ReloadManagedDatabases()

	return c.JSON(fiber.Map{"success": true, "message": "Database added successfully", "slug": slug})
}

// ─── 101% ACCURATE DB DETAILS ───────────────────────────────────────────────

type TableInfo struct {
	Name     string `json:"name"`
	RowCount int64  `json:"row_count"`
}

// GetDatabaseDetails queries actual SQLite internal schema for 101% accurate real-time data
func GetDatabaseDetails(c *fiber.Ctx) error {
	slug := c.Params("slug")

	if database.Router == nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Router not initialized"})
	}

	targetDB := database.Router.GetManagedDBBySlug(slug)
	if targetDB == nil {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Database not found or offline"})
	}

	// 1. Measure live latency
	start := time.Now()
	err := targetDB.Ping()
	latency := time.Since(start).Milliseconds()

	status := "Healthy"
	if err != nil {
		status = "Unreachable"
	}

	// 2. Query actual tables
	rows, err := targetDB.Query("SELECT name FROM sqlite_schema WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	var tables []TableInfo
	
	if err == nil {
		defer rows.Close()
		var tableNames []string
		for rows.Next() {
			var tName string
			rows.Scan(&tName)
			tableNames = append(tableNames, tName)
		}

		// 3. Count rows exactly for each table
		for _, tName := range tableNames {
			var count int64
			// SQL Injection safe here because tName comes from sqlite_schema
			targetDB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", tName)).Scan(&count)
			tables = append(tables, TableInfo{Name: tName, RowCount: count})
		}
	}

	// Calculate total size if possible (approximate via PRAGMA page_count * page_size)
	var pageSize, pageCount int64
	targetDB.QueryRow("PRAGMA page_size").Scan(&pageSize)
	targetDB.QueryRow("PRAGMA page_count").Scan(&pageCount)
	totalBytes := pageSize * pageCount

	return c.JSON(fiber.Map{
		"success": true,
		"slug": slug,
		"status": status,
		"latency_ms": latency,
		"total_bytes": totalBytes,
		"tables": tables,
	})
}

// GetPulseHealth returns real-time health data for all shards
func GetPulseHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"pulse":   GetPulseStatus(),
	})
}
