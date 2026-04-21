package database

import (
	"database/sql"
	"fmt"
)

// AuthSchema defines tables for identity and session management
const AuthSchema = `
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);
`

// InitAuthSchema applies the auth schema to the auth shard
func InitAuthSchema(db *sql.DB) error {
	if err := ExecuteSchema(db, AuthSchema); err != nil {
		return fmt.Errorf("failed to apply auth schema: %w", err)
	}
	return nil
}
