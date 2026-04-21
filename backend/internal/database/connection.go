package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// ...

// ExecuteSchema runs a multi-statement SQL string by splitting it into individual commands.
// This ensures that all tables and indexes are created properly in SQLite/LibSQL.
func ExecuteSchema(db *sql.DB, schema string) error {
	statements := strings.Split(schema, ";")
	for _, stmt := range statements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		if _, err := db.Exec(trimmed); err != nil {
			return fmt.Errorf("failed to execute statement [%s]: %w", trimmed, err)
		}
	}
	return nil
}

// ConnectTurso establishes a connection to a Turso (libsql) database
func ConnectTurso(url, token string) (*sql.DB, error) {
	connStr := fmt.Sprintf("%s?authToken=%s", url, token)
	db, err := sql.Open("libsql", connStr)
	if err != nil {
		return nil, err
	}
	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping failed: %w", err)
	}
	return db, nil
}
