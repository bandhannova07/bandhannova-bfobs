package database

import (
	"database/sql"
	"fmt"
)

// UserSchema defines tables for the full BandhanNova Ecosystem profile
const UserSchema = `
-- Core Ecosystem Profile
CREATE TABLE IF NOT EXISTS ecosystem_profiles (
    user_id TEXT PRIMARY KEY,
    full_name TEXT,
    avatar_url TEXT,
    role TEXT DEFAULT 'user',
    plan_type TEXT DEFAULT 'free',
    account_status TEXT DEFAULT 'active',
    is_verified INTEGER DEFAULT 0,
    joined_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- User Preferences & Settings
CREATE TABLE IF NOT EXISTS user_settings (
    user_id TEXT PRIMARY KEY,
    theme TEXT DEFAULT 'dark',
    language TEXT DEFAULT 'en',
    timezone TEXT DEFAULT 'UTC',
    FOREIGN KEY (user_id) REFERENCES ecosystem_profiles(user_id)
);

-- Resource Quotas & Usage Tracking
CREATE TABLE IF NOT EXISTS user_quotas (
    user_id TEXT PRIMARY KEY,
    daily_api_calls INTEGER DEFAULT 0,
    monthly_api_calls INTEGER DEFAULT 0,
    credits REAL DEFAULT 0.0,
    last_reset_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES ecosystem_profiles(user_id)
);

-- Ecosystem Metadata (Security Logs per User)
CREATE TABLE IF NOT EXISTS user_security_logs (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    ip_address TEXT,
    device_info TEXT,
    timestamp INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES ecosystem_profiles(user_id)
);

-- Legacy Data (Chat & Items)
CREATE TABLE IF NOT EXISTS user_data (
    user_id TEXT PRIMARY KEY,
    data_json TEXT,
    updated_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES ecosystem_profiles(user_id)
);

CREATE TABLE IF NOT EXISTS chat_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    message_json TEXT,
    timestamp INTEGER,
    FOREIGN KEY (user_id) REFERENCES ecosystem_profiles(user_id)
);

CREATE TABLE IF NOT EXISTS saved_items (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    item_type TEXT,
    content_json TEXT,
    timestamp INTEGER,
    FOREIGN KEY (user_id) REFERENCES ecosystem_profiles(user_id)
);
`

// InitUserSchema applies the user ecosystem schema to a user shard
func InitUserSchema(db *sql.DB) error {
	_, err := db.Exec(UserSchema)
	if err != nil {
		return fmt.Errorf("failed to apply user schema: %w", err)
	}
	return nil
}
