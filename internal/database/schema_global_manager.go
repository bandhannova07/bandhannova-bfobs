package database

import (
	"database/sql"
	"fmt"
)

// GlobalManagerSchema defines tables for system-wide configuration
const GlobalManagerSchema = `
CREATE TABLE IF NOT EXISTS managed_databases (
    slug TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    url TEXT NOT NULL,
    encrypted_token TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
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

// InitGlobalManagerSchema applies the global manager schema to the global shard
func InitGlobalManagerSchema(db *sql.DB) error {
	_, err := db.Exec(GlobalManagerSchema)
	if err != nil {
		return fmt.Errorf("failed to apply global manager schema: %w", err)
	}
	return nil
}
