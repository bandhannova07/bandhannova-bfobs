package database

import (
	"database/sql"
	"fmt"
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

CREATE TABLE IF NOT EXISTS managed_api_keys (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    encrypted_value TEXT NOT NULL,
    label TEXT,
    status TEXT DEFAULT 'active',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
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
	
	// Add product_id to managed_databases if missing
	_, _ = db.Exec("ALTER TABLE managed_databases ADD COLUMN product_id TEXT")
	
	// Add new columns to managed_products if missing
	_, _ = db.Exec("ALTER TABLE managed_products ADD COLUMN app_type TEXT DEFAULT 'website'")
	_, _ = db.Exec("ALTER TABLE managed_products ADD COLUMN app_url TEXT")
	_, _ = db.Exec("ALTER TABLE managed_products ADD COLUMN url TEXT") // Keep for safety if already existed

	return nil
}
