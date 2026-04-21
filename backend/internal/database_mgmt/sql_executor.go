package database_mgmt

import (
	"database/sql"
	"fmt"
	"github.com/bandhannova/api-hunter/internal/database"
)

type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Message string                   `json:"message"`
}

// ExecuteSQL runs a raw SQL query on a specific database shard
func ExecuteSQL(shardSlug string, query string) (*QueryResult, error) {
	db := database.Router.GetManagedDBBySlug(shardSlug)
	if db == nil {
		return nil, fmt.Errorf("shard not found: %s", shardSlug)
	}

	// Try to determine if it's a SELECT or an action (INSERT/UPDATE/DELETE)
	// Very basic check, should be more robust in production
	isSelect := (query[:6] == "SELECT" || query[:6] == "select")

	if isSelect {
		return runQuery(db, query)
	} else {
		return runExec(db, query)
	}
}

func runQuery(db *sql.DB, query string) (*QueryResult, error) {
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var resultRows []map[string]interface{}
	for rows.Next() {
		// Prepare pointers for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			var v interface{}
			val := values[i]
			b, ok := val.([]byte)
			if ok {
				v = string(b)
			} else {
				v = val
			}
			rowMap[col] = v
		}
		resultRows = append(resultRows, rowMap)
	}

	return &QueryResult{
		Columns: columns,
		Rows:    resultRows,
		Message: fmt.Sprintf("%d rows returned", len(resultRows)),
	}, nil
}

func runExec(db *sql.DB, query string) (*QueryResult, error) {
	res, err := db.Exec(query)
	if err != nil {
		return nil, err
	}

	affected, _ := res.RowsAffected()
	return &QueryResult{
		Columns: []string{},
		Rows:    []map[string]interface{}{},
		Message: fmt.Sprintf("Success: %d rows affected", affected),
	}, nil
}
