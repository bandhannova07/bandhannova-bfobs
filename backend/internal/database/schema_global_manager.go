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
    master_schema TEXT,
    access_token TEXT,
    gateway_code TEXT,
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
    endpoint_url TEXT,
    platform_type TEXT DEFAULT 'openai_compatible',
    limit_rps INTEGER DEFAULT 0,
    limit_rpm INTEGER DEFAULT 0,
    limit_rph INTEGER DEFAULT 0,
    limit_rpd INTEGER DEFAULT 0,
    limit_rpmonth INTEGER DEFAULT 0,
    limit_concurrent INTEGER DEFAULT 0,
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

CREATE TABLE IF NOT EXISTS api_usage_logs (
    id TEXT PRIMARY KEY,
    key_id TEXT NOT NULL,
    card_id TEXT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    status_code INTEGER,
    latency_ms INTEGER,
    ip_address TEXT,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (key_id) REFERENCES managed_api_keys(id),
    FOREIGN KEY (card_id) REFERENCES api_cards(id)
);

CREATE TABLE IF NOT EXISTS storage_buckets (
    id TEXT PRIMARY KEY,
    product_id TEXT NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    is_public INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (product_id) REFERENCES managed_products(id) ON DELETE CASCADE,
    UNIQUE(product_id, slug)
);


`

// InitGlobalManagerSchema applies the global manager schema and handles migrations
func InitGlobalManagerSchema(db *sql.DB) error {
	if err := ExecuteSchema(db, GlobalManagerSchema); err != nil {
		return fmt.Errorf("failed to apply global manager schema: %w", err)
	}

	log.Println("🛠️  Running API Management migrations...")

	// Ensure api_cards has new columns if it existed before
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN endpoint_url TEXT")
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN platform_type TEXT DEFAULT 'openai_compatible'")
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN limit_rps INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN limit_rpm INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN limit_rph INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN limit_rpd INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN limit_rpmonth INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE api_cards ADD COLUMN limit_concurrent INTEGER DEFAULT 0")
	_, _ = db.Exec("ALTER TABLE managed_products ADD COLUMN master_schema TEXT")
	_, _ = db.Exec("ALTER TABLE managed_products ADD COLUMN access_token TEXT")
	_, _ = db.Exec("ALTER TABLE managed_products ADD COLUMN gateway_code TEXT")

	// Ensure managed_api_keys has new columns
	_, _ = db.Exec("ALTER TABLE managed_api_keys ADD COLUMN card_id TEXT")
	_, _ = db.Exec("ALTER TABLE managed_api_keys ADD COLUMN is_deleted INTEGER DEFAULT 0")

	// ─── Performance Indexes ─────────────────────────────────────────────────────
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_usage_logs_card_ts ON api_usage_logs(card_id, timestamp)")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_usage_logs_key_ts ON api_usage_logs(key_id, timestamp)")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_keys_card_status ON managed_api_keys(card_id, status, is_deleted)")
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_cards_section ON api_cards(section_id, is_deleted)")

	log.Println("✅ API Management migrations completed")
	return nil
}

