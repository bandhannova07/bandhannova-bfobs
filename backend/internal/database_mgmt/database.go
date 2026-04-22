package database_mgmt

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/bandhannova/api-hunter/internal/config"
	"github.com/bandhannova/api-hunter/internal/database"
	"github.com/bandhannova/api-hunter/internal/security"
	"github.com/bandhannova/api-hunter/internal/storage_mgmt"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type DatabaseResponse struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	URL       string `json:"db_url"`
	ProductID string `json:"product_id,omitempty"`
	Status    string `json:"status"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	IsCore    bool   `json:"is_core"` // True if loaded from .env
}

type ProductResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	AppType      string `json:"app_type"`
	AppURL       string `json:"app_url"`
	Description  string `json:"description"`
	Icon         string `json:"icon"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	GatewayCode  string `json:"gateway_code,omitempty"`
	Status       string `json:"status"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// ReloadManagedDatabases hot-swaps active DBs from the global managed_databases table
func ReloadManagedDatabases() error {
	if database.Router == nil {
		return fmt.Errorf("shard router not initialized")
	}

	var mDBs []database.ManagedDB
	var mu sync.Mutex
	var wg sync.WaitGroup


	// Actually, let's just use the router's globalManagerDBs slice
	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			rows, err := db.Query("SELECT id, slug, name, category, db_url, encrypted_token FROM managed_databases WHERE status = 'active'")
			if err != nil {
				return
			}
			defer rows.Close()

			for rows.Next() {
				var id, slug, name, category, dbURL, encrypted string
				if err := rows.Scan(&id, &slug, &name, &category, &dbURL, &encrypted); err != nil {
					continue
				}

				token, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
				if err != nil {
					continue
				}

				connStr := fmt.Sprintf("%s?authToken=%s", dbURL, token)
				targetDB, err := sql.Open("libsql", connStr)
				if err != nil {
					continue
				}

				mu.Lock()
				mDBs = append(mDBs, database.ManagedDB{
					Slug:     slug,
					Name:     name,
					Category: category,
					DB:       targetDB,
				})
				mu.Unlock()
			}
		}(gDB)
	}

	wg.Wait()
	database.Router.ReloadDynamicDBs(mDBs)
	return nil
}

// ReloadManagedAPIKeys hot-reloads API keys from the global managed_api_keys table
func ReloadManagedAPIKeys() error {
	if database.Router == nil {
		return fmt.Errorf("shard router not initialized")
	}

	providerKeys := make(map[string][]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			rows, err := db.Query("SELECT provider, encrypted_value FROM managed_api_keys WHERE status = 'active'")
			if err != nil {
				return
			}
			defer rows.Close()

			for rows.Next() {
				var provider, encrypted string
				if err := rows.Scan(&provider, &encrypted); err != nil {
					continue
				}

				key, err := security.Decrypt(encrypted, config.AppConfig.BandhanNovaMasterKey)
				if err != nil {
					continue
				}

				mu.Lock()
				providerKeys[provider] = append(providerKeys[provider], key)
				mu.Unlock()
			}
		}(gDB)
	}

	wg.Wait()

	for provider, keys := range providerKeys {
		config.UpdateFleetKeys(provider, keys)
	}

	log.Printf("🛡️  Managed API keys reloaded from all global shards")
	return nil
}

// ListDatabases returns all active databases (core + managed)
func ListDatabases(c *fiber.Ctx) error {
	productID := c.Query("product_id")
	var resp []DatabaseResponse

	// 1. Core Databases (Only if NO product_id is provided)
	if productID == "" {
		// Core Master Shard
		if config.AppConfig.TursoCoreURL != "" {
			resp = append(resp, DatabaseResponse{ID: "core-master", Slug: "core-master", Name: "Core Master (Shard 1)", Category: "core", URL: config.AppConfig.TursoCoreURL, Status: "active", IsCore: true})
		}
		// Global Manager Shards
		for i, u := range config.AppConfig.TursoGlobalURLs {
			if u != "" {
				slug := fmt.Sprintf("core-gm-%d", i)
				name := fmt.Sprintf("Global Manager (Shard %d)", i+2)
				resp = append(resp, DatabaseResponse{ID: slug, Slug: slug, Name: name, Category: "global", URL: u, Status: "active", IsCore: true})
			}
		}
		// Auth & Analytics Shards
		if config.AppConfig.TursoAuthURL != "" {
			resp = append(resp, DatabaseResponse{ID: "core-auth", Slug: "core-auth", Name: "Core Auth", Category: "auth", URL: config.AppConfig.TursoAuthURL, Status: "active", IsCore: true})
		}
		if config.AppConfig.TursoAnalyticsURL != "" {
			resp = append(resp, DatabaseResponse{ID: "core-analytics", Slug: "core-analytics", Name: "Core Analytics", Category: "analytics", URL: config.AppConfig.TursoAnalyticsURL, Status: "active", IsCore: true})
		}
		
		for i, u := range config.AppConfig.TursoUserShardURLs {
			if u != "" {
				slug := fmt.Sprintf("core-user-%d", i)
				resp = append(resp, DatabaseResponse{ID: slug, Slug: slug, Name: fmt.Sprintf("Core User Shard %d", i), Category: "user", URL: u, Status: "active", IsCore: true})
			}
		}
	}

	// 2. Add Managed DBs (From all Global Shards)
	for _, db := range database.Router.GetAllGlobalManagerDBs() {
		query := "SELECT id, slug, name, category, db_url, product_id, status, created_at, updated_at FROM managed_databases"
		var rows *sql.Rows
		var err error

		if productID != "" {
			rows, err = db.Query(query+" WHERE product_id = ? ORDER BY created_at DESC", productID)
		} else {
			rows, err = db.Query(query + " ORDER BY created_at DESC")
		}

		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var d DatabaseResponse
				var pID sql.NullString
				if err := rows.Scan(&d.ID, &d.Slug, &d.Name, &d.Category, &d.URL, &pID, &d.Status, &d.CreatedAt, &d.UpdatedAt); err == nil {
					d.IsCore = false
					if pID.Valid {
						d.ProductID = pID.String
					}
					resp = append(resp, d)
				}
			}
		}
	}

	return c.JSON(fiber.Map{"success": true, "databases": resp})
}

type AddDBRequest struct {
	Name      string `json:"name"`
	Category  string `json:"category"`
	URL       string `json:"db_url"`
	Token     string `json:"token"`
	ProductID string `json:"product_id"`
}

// AddDatabase adds a new dynamic database with auto-indexing naming
func AddDatabase(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req AddDBRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	if req.Category == "" || req.URL == "" || req.Token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Category, URL, and Token required"})
	}

	validCats := map[string]string{
		"auth":      "Auth Shard",
		"analytics": "Analytics Shard",
		"global":    "Global Manager",
		"user":      "User Shard",
	}

	baseName, ok := validCats[req.Category]
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Invalid category"})
	}

	// 1. Calculate Auto-Index for Name
	var count int
	err := database.Router.GetGlobalManagerDB().QueryRow(
		"SELECT COUNT(*) FROM managed_databases WHERE category = ?", 
		req.Category,
	).Scan(&count)
	if err != nil {
		count = 0
	}

	// Set final name: e.g., "User Shard 0"
	finalName := fmt.Sprintf("%s %d", baseName, count)
	if req.Name != "" {
		// If user provided a name, we can still use it or override.
		// User requested auto, so we prioritize the generated one.
		finalName = req.Name 
	}
	// 1. Check for Duplicate URL
	var exists int
	database.Router.GetGlobalManagerDB().QueryRow(
		"SELECT COUNT(*) FROM managed_databases WHERE db_url = ?", 
		req.URL,
	).Scan(&exists)
	if exists > 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Database with this URL already exists in system"})
	}

	// 2. Test Connection
	testDB, err := database.ConnectTurso(req.URL, req.Token)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": true, "message": "Failed to connect to Turso database. Verify URL and Token."})
	}
	defer testDB.Close()

	// 3. Encrypt Token
	encrypted, err := security.Encrypt(req.Token, config.AppConfig.BandhanNovaMasterKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Encryption failed"})
	}

	// 4. Generate Slug & Save
	slug := strings.ToLower(strings.ReplaceAll(finalName, " ", "-")) + "-" + uuid.New().String()[:6]
	id := uuid.New().String()
	now := time.Now().Unix()

	_, err = database.Router.GetGlobalManagerDB().Exec(
		"INSERT INTO managed_databases (id, slug, name, category, db_url, encrypted_token, product_id, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)",
		id, slug, finalName, req.Category, req.URL, encrypted, req.ProductID, now, now,
	)
	if err != nil {
		log.Printf("Failed to save DB: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": true, "message": "Failed to save database config"})
	}

	logAudit("ADD_DATABASE", req.Category, ip, fmt.Sprintf("Added DB: %s (%s)", finalName, req.URL))

	// Hot Reload
	go ReloadManagedDatabases()

	return c.JSON(fiber.Map{"success": true, "message": "Database added successfully", "slug": slug})
}

// HarmonizeNames renames all existing database records to the indexed format
func HarmonizeNames() error {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return fmt.Errorf("global DB not connected")
	}

	categories := []string{"auth", "analytics", "global", "user"}
	validCats := map[string]string{
		"auth":      "Auth Shard",
		"analytics": "Analytics Shard",
		"global":    "Global Manager",
		"user":      "User Shard",
	}

	for _, cat := range categories {
		rows, err := database.Router.GetGlobalManagerDB().Query(
			"SELECT id, db_url FROM managed_databases WHERE category = ? ORDER BY created_at ASC", 
			cat,
		)
		if err != nil {
			continue
		}
		
		type item struct{ id, url string }
		var items []item
		for rows.Next() {
			var id, url string
			rows.Scan(&id, &url)
			items = append(items, item{id, url})
		}
		rows.Close()

		for i, itm := range items {
			newName := fmt.Sprintf("%s %d", validCats[cat], i)
			newSlug := strings.ToLower(strings.ReplaceAll(newName, " ", "-")) + "-" + itm.id[:6]
			
			database.Router.GetGlobalManagerDB().Exec(
				"UPDATE managed_databases SET name = ?, slug = ? WHERE id = ?",
				newName, newSlug, itm.id,
			)
		}
	}
	
	log.Println("🎨 Database names harmonized with new indexing system")
	return nil
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

// ─── PRODUCT MANAGEMENT ───────────────────────────────────────────────────

func ListProducts(c *fiber.Ctx) error {
	if database.Router == nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Router not initialized"})
	}

	var products []ProductResponse
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			rows, err := db.Query(`
				SELECT p.id, p.name, p.slug, p.app_type, p.app_url, p.description, p.icon, p.status, p.created_at, p.updated_at, c.client_id, c.client_secret
				FROM managed_products p
				LEFT JOIN oauth_clients c ON p.id = c.product_id
				ORDER BY p.created_at DESC
			`)
			if err != nil {
				return
			}
			defer rows.Close()

			for rows.Next() {
				var p ProductResponse
				var appType, appURL, icon, clientID, clientSecret sql.NullString
				if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &appType, &appURL, &p.Description, &icon, &p.Status, &p.CreatedAt, &p.UpdatedAt, &clientID, &clientSecret); err == nil {
					if appType.Valid { p.AppType = appType.String }
					if appURL.Valid { p.AppURL = appURL.String }
					if icon.Valid { p.Icon = icon.String }
					if clientID.Valid { p.ClientID = clientID.String }
					if clientSecret.Valid { p.ClientSecret = clientSecret.String }
					
					mu.Lock()
					products = append(products, p)
					mu.Unlock()
				}
			}
		}(gDB)
	}

	wg.Wait()
	return c.JSON(fiber.Map{"success": true, "products": products})
}

// GetProductDetails fetches a single product by its slug across all shards
func GetProductDetails(c *fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Slug is required"})
	}

	var p ProductResponse
	var appType, appURL, icon, clientID, clientSecret sql.NullString
	found := false

	// Search all Global Manager shards in parallel
	var mu sync.Mutex
	var wg sync.WaitGroup
	found = false

	for _, gDB := range database.Router.GetAllGlobalManagerDBs() {
		wg.Add(1)
		go func(db *sql.DB) {
			defer wg.Done()
			
			var localP ProductResponse
			var localAppType, localAppURL, localIcon, localClientID, localClientSecret, localAccessToken, localGatewayCode sql.NullString
			
			err := db.QueryRow(`
				SELECT p.id, p.name, p.slug, p.app_type, p.app_url, p.description, p.icon, p.access_token, p.gateway_code, p.status, p.created_at, p.updated_at, c.client_id, c.client_secret
				FROM managed_products p
				LEFT JOIN oauth_clients c ON p.id = c.product_id
				WHERE p.slug = ?`,
				slug,
			).Scan(&localP.ID, &localP.Name, &localP.Slug, &localAppType, &localAppURL, &localP.Description, &localIcon, &localAccessToken, &localGatewayCode, &localP.Status, &localP.CreatedAt, &localP.UpdatedAt, &localClientID, &localClientSecret)

			if err == nil {
				mu.Lock()
				if !found {
					p = localP
					appType = localAppType
					appURL = localAppURL
					icon = localIcon
					clientID = localClientID
					clientSecret = localClientSecret
					if localAccessToken.Valid { p.AccessToken = localAccessToken.String }
					if localGatewayCode.Valid { p.GatewayCode = localGatewayCode.String }
					found = true
				}
				mu.Unlock()
			}
		}(gDB)
	}
	wg.Wait()

	if !found {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Infrastructure not found on any shard"})
	}

	if appType.Valid { p.AppType = appType.String }
	if appURL.Valid { p.AppURL = appURL.String }
	if icon.Valid { p.Icon = icon.String }
	if clientID.Valid { p.ClientID = clientID.String }
	if clientSecret.Valid { p.ClientSecret = clientSecret.String }

	return c.JSON(fiber.Map{"success": true, "product": p})
}

// RemoveDatabase deletes a database shard from the system
func RemoveDatabase(c *fiber.Ctx) error {
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

	ip := c.IP()
	tx, err := database.Router.GetGlobalManagerDB().Begin()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Transaction failed"})
	}

	_, err = tx.Exec("DELETE FROM managed_databases WHERE id = ?", id)
	if err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to delete shard"})
	}

	tx.Commit()
	logAudit("REMOVE_SHARD", id, ip, fmt.Sprintf("Removed database shard ID: %s", id))

	return c.JSON(fiber.Map{"success": true, "message": "Shard decommissioned successfully"})
}

// AddProduct handles creating a new product with automated OAuth and Storage provisioning
type AddProductRequest struct {
	Name        string `json:"name"`
	AppType     string `json:"app_type"`
	AppURL      string `json:"app_url"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func AddProduct(c *fiber.Ctx) error {
	ip, _ := c.Locals("admin_ip").(string)
	var req AddProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Product name is required"})
	}

	id := uuid.New().String()
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-"))
	now := time.Now().Unix()

	// 1. Resolve target Global Manager shard (Load Balanced)
	targetGDB := database.Router.GetGlobalManagerDBBySlug(slug)
	if targetGDB == nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "No Global Manager shards available"})
	}

	// Auto-generate OAuth Credentials
	clientID := "bn_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	clientSecret := base64.StdEncoding.EncodeToString([]byte(uuid.New().String() + uuid.New().String()))[:48]

	// Auto-generate Access Token (API Key for product-level auth)
	accessToken := "bfobs_" + strings.ReplaceAll(uuid.New().String(), "-", "") + strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	// Auto-generate Gateway Code (random URL-safe code for bdn-bfobs:// URL)
	gatewayCode := strings.ReplaceAll(uuid.New().String(), "-", "")[:12]

	tx, err := targetGDB.Begin()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Transaction failed"})
	}

	_, err = tx.Exec(
		"INSERT INTO managed_products (id, name, slug, app_type, app_url, description, icon, access_token, gateway_code, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)",
		id, req.Name, slug, req.AppType, req.AppURL, req.Description, req.Icon, accessToken, gatewayCode, now, now,
	)
	if err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to save product"})
	}

	_, err = tx.Exec(
		"INSERT INTO oauth_clients (client_id, client_secret, product_id, redirect_uris, created_at) VALUES (?, ?, ?, ?, ?)",
		clientID, clientSecret, id, "[]", now,
	)
	if err != nil {
		tx.Rollback()
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to create OAuth client"})
	}

	tx.Commit()

	// Automatic storage
	go storage_mgmt.InitializeProductFolder(slug)
	
	logAuditShard(targetGDB, "ADD_PRODUCT", req.Name, ip, fmt.Sprintf("Added product: %s (Client: %s)", req.Name, clientID))

	return c.JSON(fiber.Map{
		"success":       true,
		"message":       "Product added with OAuth credentials",
		"id":            id,
		"client_id":     clientID,
		"client_secret": clientSecret,
		"access_token":  accessToken,
		"gateway_url":   fmt.Sprintf("bdn-bfobs://%s/%s/gateway/", slug, gatewayCode),
	})
}

func UpdateProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	ip, _ := c.Locals("admin_ip").(string)
	var req AddProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	now := time.Now().Unix()
	found := false
	
	for _, db := range database.Router.GetAllGlobalManagerDBs() {
		res, err := db.Exec(
			"UPDATE managed_products SET name = ?, app_type = ?, app_url = ?, description = ?, icon = ?, updated_at = ? WHERE id = ?",
			req.Name, req.AppType, req.AppURL, req.Description, req.Icon, now, id,
		)
		if err == nil {
			rows, _ := res.RowsAffected()
			if rows > 0 {
				found = true
				break
			}
		}
	}

	if !found {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Product not found on any shard"})
	}

	logAudit("UPDATE_PRODUCT", req.Name, ip, fmt.Sprintf("Updated product: %s", req.Name))
	return c.JSON(fiber.Map{"success": true, "message": "Product updated successfully"})
}

type DeleteProductRequest struct {
	MasterKey    string `json:"master_key"`
	Confirmation string `json:"confirmation"`
}

func DeleteProduct(c *fiber.Ctx) error {
	id := c.Params("id")
	ip, _ := c.Locals("admin_ip").(string)
	var req DeleteProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Invalid payload"})
	}

	// 1. Verify Master Key
	if req.MasterKey != config.AppConfig.BandhanNovaMasterKey {
		return c.Status(401).JSON(fiber.Map{"error": true, "message": "Invalid Master Key"})
	}

	// 2. Get Product Info
	var pName string
	var pNameFound bool
	for _, db := range database.Router.GetAllGlobalManagerDBs() {
		err := db.QueryRow("SELECT name FROM managed_products WHERE id = ?", id).Scan(&pName)
		if err == nil {
			pNameFound = true
			break
		}
	}
	
	if !pNameFound {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Product not found"})
	}

	// 3. Verify Phrase
	expectedPhrase := fmt.Sprintf("I am Bandhan, to the best of my knowledge, I want to delete this product, named %s.", pName)
	if req.Confirmation != expectedPhrase {
		return c.Status(400).JSON(fiber.Map{"error": true, "message": "Confirmation phrase mismatch"})
	}

	// 4. Find linked shards
	rows, err := database.Router.GetGlobalManagerDB().Query("SELECT slug, name FROM managed_databases WHERE product_id = ?", id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var slug, shardName string
			if err := rows.Scan(&slug, &shardName); err == nil {
				// WIPE SHARD DATA
				targetDB := database.Router.GetManagedDBBySlug(slug)
				if targetDB != nil {
					// Drop all tables
					tRows, _ := targetDB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
					if tRows != nil {
						var tables []string
						for tRows.Next() {
							var tName string
							tRows.Scan(&tName)
							tables = append(tables, tName)
						}
						tRows.Close()
						for _, tName := range tables {
							targetDB.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", tName))
						}
						targetDB.Exec("VACUUM")
					}
				}
				// Set product_id to NULL (Moves to Unused Shards)
				database.Router.GetGlobalManagerDB().Exec("UPDATE managed_databases SET product_id = NULL WHERE slug = ?", slug)
			}
		}
	}

	// 5. Delete Product
	var deleteFound bool
	for _, db := range database.Router.GetAllGlobalManagerDBs() {
		res, err := db.Exec("DELETE FROM managed_products WHERE id = ?", id)
		if err == nil {
			count, _ := res.RowsAffected()
			if count > 0 {
				// Also delete OAuth client from the same shard
				db.Exec("DELETE FROM oauth_clients WHERE product_id = ?", id)
				// And cleanup managed_databases on THIS shard if they exist
				db.Exec("DELETE FROM managed_databases WHERE product_id = ?", id)
				deleteFound = true
				break
			}
		}
	}

	if !deleteFound {
		return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to delete product record from fleet"})
	}

	logAudit("DELETE_PRODUCT", pName, ip, fmt.Sprintf("DELETED PRODUCT AND WIPED SHARDS: %s", pName))
	
	// Refresh Registry
	go ReloadManagedDatabases()

	return c.JSON(fiber.Map{"success": true, "message": "Product deleted and shards wiped successfully"})
}

func ResetOAuthCredentials(c *fiber.Ctx) error {
	id := c.Params("id")
	ip, _ := c.Locals("admin_ip").(string)

	now := time.Now().Unix()
	clientID := "bn_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	clientSecret := base64.StdEncoding.EncodeToString([]byte(uuid.New().String() + uuid.New().String()))[:48]

	var resetFound bool
	for _, db := range database.Router.GetAllGlobalManagerDBs() {
		// Check if product exists on this shard
		var exists int
		db.QueryRow("SELECT 1 FROM managed_products WHERE id = ?", id).Scan(&exists)
		if exists == 1 {
			tx, err := db.Begin()
			if err != nil {
				continue
			}
			// Delete old if exists
			tx.Exec("DELETE FROM oauth_clients WHERE product_id = ?", id)
			// Insert new
			_, err = tx.Exec(
				"INSERT INTO oauth_clients (client_id, client_secret, product_id, redirect_uris, created_at) VALUES (?, ?, ?, ?, ?)",
				clientID, clientSecret, id, "[]", now,
			)
			if err != nil {
				tx.Rollback()
				return c.Status(500).JSON(fiber.Map{"error": true, "message": "Failed to reset OAuth on target shard"})
			}
			tx.Commit()
			resetFound = true
			break
		}
	}

	if !resetFound {
		return c.Status(404).JSON(fiber.Map{"error": true, "message": "Product not found to reset OAuth"})
	}

	logAudit("RESET_OAUTH", id, ip, fmt.Sprintf("Reset OAuth credentials for product ID: %s", id))
	return c.JSON(fiber.Map{"success": true, "client_id": clientID, "client_secret": clientSecret})
}

// GetPulseHealth returns real-time health data for all shards
func GetPulseHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"pulse":   GetPulseStatus(),
	})
}

func logAuditShard(db *sql.DB, action, target, ip, details string) {
	if db == nil {
		return
	}
	_, err := db.Exec(
		"INSERT INTO admin_audit_logs (action, target, ip, details, timestamp) VALUES (?, ?, ?, ?, ?)",
		action, target, ip, details, time.Now().Unix(),
	)
	if err != nil {
		log.Printf("Failed to log audit: %v", err)
	}
}

func logAudit(action, target, ip, details string) {
	if database.Router == nil || database.Router.GetGlobalManagerDB() == nil {
		return
	}
	logAuditShard(database.Router.GetGlobalManagerDB(), action, target, ip, details)
}

func createHFRepoInternal(name string, private bool) {
	token := config.AppConfig.HFToken
	if token == "" {
		log.Println("HF_TOKEN missing, skipping auto-storage creation")
		return
	}

	apiUrl := "https://huggingface.co/api/repos/create"
	payload := map[string]interface{}{
		"name":    name,
		"type":    "dataset",
		"private": private,
	}
	
	jsonPayload, _ := json.Marshal(payload)
	hReq, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonPayload))
	hReq.Header.Set("Authorization", "Bearer "+token)
	hReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(hReq)
	if err != nil {
		log.Printf("Failed to auto-create HF repo: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("HF Auto-Provisioning: %s (Status: %d)", name, resp.StatusCode)
}
