package database

import (
	"database/sql"
	"fmt"
	"log"
)

// InfrastructureSchema defines tables for managing the fleet's shards and resources
const InfrastructureSchema = `
CREATE TABLE IF NOT EXISTS infrastructure_shards (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- 'global_manager', 'auth', 'analytics', 'user'
    db_url TEXT NOT NULL,
    encrypted_token TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS infrastructure_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);
`

// InitInfrastructureSchema applies the core infrastructure schema
func InitInfrastructureSchema(db *sql.DB) error {
	_, err := db.Exec(InfrastructureSchema)
	if err != nil {
		return fmt.Errorf("failed to apply infrastructure schema: %w", err)
	}
	log.Println("✨ Core Infrastructure schema applied")
	return nil
}
