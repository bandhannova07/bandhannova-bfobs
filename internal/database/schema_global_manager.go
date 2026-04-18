package database

import (
	"database/sql"
	"fmt"
	"log"
)

// GlobalManagerSchema defines tables for system-wide configuration
const GlobalManagerSchema = `
CREATE TABLE IF NOT EXISTS managed_products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    app_type TEXT DEFAULT 'website',
    app_url TEXT,
    description TEXT,
    icon TEXT,
    status TEXT DEFAULT 'active',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS managed_databases (
    id TEXT PRIMARY KEY,
    slug TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    db_url TEXT NOT NULL,
    encrypted_token TEXT NOT NULL,
    product_id TEXT,
    status TEXT DEFAULT 'active',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY(product_id) REFERENCES managed_products(id)
);

CREATE TABLE IF NOT EXISTS oauth_clients (
    client_id TEXT PRIMARY KEY,
    client_secret TEXT NOT NULL,
    product_id TEXT NOT NULL,
    redirect_uris TEXT NOT NULL, -- JSON array of strings
    grants TEXT DEFAULT '["authorization_code", "refresh_token"]',
    created_at INTEGER NOT NULL,
    FOREIGN KEY (product_id) REFERENCES managed_products(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS oauth_authorizations (
    code TEXT PRIMARY KEY,
    client_id TEXT NOT NULL,
    user_id TEXT NOT NULL, -- User ID from Shards
    scope TEXT,
    expires_at INTEGER NOT NULL,
    redirect_uri TEXT,
    FOREIGN KEY (client_id) REFERENCES oauth_clients(client_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS api_sections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS api_cards (
    id TEXT PRIMARY KEY,
    section_id TEXT NOT NULL,
    name TEXT NOT NULL,
    icon TEXT,
    description TEXT,
    is_deleted INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (section_id) REFERENCES api_sections(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS managed_api_keys (
    id TEXT PRIMARY KEY,
    card_id TEXT, -- Linked to api_cards
    provider TEXT NOT NULL, -- Keep for legacy/routing
    encrypted_value TEXT NOT NULL,
    label TEXT,
    api_url TEXT, -- Custom endpoint if use_url is true
    use_url INTEGER DEFAULT 0, -- 0 for false, 1 for true
    status TEXT DEFAULT 'active',
    is_deleted INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (card_id) REFERENCES api_cards(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS admin_audit_log (
    id TEXT PRIMARY KEY,
    action TEXT NOT NULL,
    target TEXT NOT NULL,
    ip_address TEXT,
    details TEXT,
    timestamp INTEGER NOT NULL
);
`

// InitGlobalManagerSchema applies the global manager schema and handles migrations
func InitGlobalManagerSchema(db *sql.DB) error {
	_, err := db.Exec(GlobalManagerSchema)
	if err != nil {
		return fmt.Errorf("failed to apply global manager schema: %w", err)
	}

	// ─── AUTO-MIGRATIONS FOR EXISTING TABLES ────────────────────────────────
	
	// ─── API MANAGEMENT MIGRATIONS ───────────────────────────────────────
	log.Println("🛠️  Running API Management migrations...")
	
	// Create tables explicitly first to ensure they exist for the migrations below
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_sections (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL
		);
	`)
	if err != nil { log.Printf("⚠️  Error creating api_sections: %v", err) }

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_cards (
			id TEXT PRIMARY KEY,
			section_id TEXT NOT NULL,
			name TEXT NOT NULL,
			icon TEXT,
			description TEXT,
			is_deleted INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			FOREIGN KEY (section_id) REFERENCES api_sections(id) ON DELETE CASCADE
		);
	`)
	if err != nil { log.Printf("⚠️  Error creating api_cards: %v", err) }

	// Add columns to existing managed_api_keys table
	_, _ = db.Exec("ALTER TABLE managed_api_keys ADD COLUMN card_id TEXT")
	_, _ = db.Exec("ALTER TABLE managed_api_keys ADD COLUMN api_url TEXT")
	_, _ = db.Exec("ALTER TABLE managed_api_keys ADD COLUMN use_url INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE managed_api_keys ADD COLUMN is_deleted INTEGER DEFAULT 0")

	// Ensure "Unused APIs" section exists
	unusedID := "unused"
	_, err = db.Exec("INSERT OR IGNORE INTO api_sections (id, name, created_at) VALUES (?, ?, ?)", unusedID, "Unused APIs", 0)
	if err != nil {
		log.Printf("⚠️  Failed to seed Unused APIs section: %v", err)
	}

	log.Println("✅ API Management migrations completed")
	return nil
}
