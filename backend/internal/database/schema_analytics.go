package database

import (
	"database/sql"
	"fmt"
)

// AnalyticsSchema defines tables for tracking requests and emails
const AnalyticsSchema = `
CREATE TABLE IF NOT EXISTS request_logs (
    id TEXT PRIMARY KEY,
    method TEXT,
    path TEXT,
    status_code INTEGER,
    latency_ms INTEGER,
    ip_address TEXT,
    user_id TEXT,
    timestamp INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS inbound_emails (
    id TEXT PRIMARY KEY,
    from_email TEXT,
    to_email TEXT,
    subject TEXT,
    content_text TEXT,
    content_html TEXT,
    timestamp INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS outbound_emails (
    id TEXT PRIMARY KEY,
    to_email TEXT,
    subject TEXT,
    provider TEXT,
    status TEXT,
    timestamp INTEGER NOT NULL
);
`

// InitAnalyticsSchema applies the analytics schema to the analytics shard
func InitAnalyticsSchema(db *sql.DB) error {
	if err := ExecuteSchema(db, AnalyticsSchema); err != nil {
		return fmt.Errorf("failed to apply analytics schema: %w", err)
	}
	return nil
}
