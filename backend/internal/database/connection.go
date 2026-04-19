package database

import (
	"database/sql"
	"fmt"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

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
