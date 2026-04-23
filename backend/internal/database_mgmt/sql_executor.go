package database_mgmt

import (
	"database/sql"
	"fmt"
	"strings"
	"github.com/bandhannova/api-hunter/internal/database"
)

type QueryResult struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
	Message string                   `json:"message"`
}

// ExecuteSQL runs raw SQL (single or multi-statement) on a specific database shard.
// For multi-statement input, it splits by semicolons, executes each sequentially,
// and returns the result of the LAST statement (if it's a query) or a summary.
func ExecuteSQL(shardSlug string, query string) (*QueryResult, error) {
	db := database.Router.GetManagedDBBySlug(shardSlug)
	if db == nil {
		return nil, fmt.Errorf("shard not found: %s", shardSlug)
	}

	// Split into individual statements
	statements := splitStatements(query)
	if len(statements) == 0 {
		return nil, fmt.Errorf("no valid SQL statements found")
	}

	// Single statement — fast path
	if len(statements) == 1 {
		return executeSingle(db, statements[0])
	}

	// Multi-statement — execute each, return last result
	var lastResult *QueryResult
	totalAffected := int64(0)

	for i, stmt := range statements {
		result, err := executeSingle(db, stmt)
		if err != nil {
			return nil, fmt.Errorf("statement %d failed: %w\n→ %s", i+1, err, truncate(stmt, 80))
		}
		lastResult = result
		// Track total rows affected for non-query statements
		if result.Columns == nil || len(result.Columns) == 0 {
			// Parse affected count from message
			var affected int64
			fmt.Sscanf(result.Message, "Success: %d rows affected", &affected)
			totalAffected += affected
		}
	}

	// If the last statement was a query, return its rows
	if lastResult != nil && len(lastResult.Columns) > 0 {
		lastResult.Message = fmt.Sprintf("%s (after %d statement(s))", lastResult.Message, len(statements))
		return lastResult, nil
	}

	// Otherwise, return summary of all exec statements
	return &QueryResult{
		Columns: []string{},
		Rows:    []map[string]interface{}{},
		Message: fmt.Sprintf("Success: %d statement(s) executed, %d total rows affected", len(statements), totalAffected),
	}, nil
}

// executeSingle runs a single SQL statement and returns the result
func executeSingle(db *sql.DB, stmt string) (*QueryResult, error) {
	cleaned := stripComments(stmt)
	if len(cleaned) < 3 {
		// Skip empty/comment-only statements
		return &QueryResult{
			Columns: []string{},
			Rows:    []map[string]interface{}{},
			Message: "Skipped empty statement",
		}, nil
	}

	upper := strings.ToUpper(cleaned)
	isQuery := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "PRAGMA") ||
		strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "EXPLAIN") ||
		strings.HasPrefix(upper, "WITH")

	if isQuery {
		return runQuery(db, stmt)
	}
	return runExec(db, stmt)
}

// splitStatements splits SQL text into individual statements by semicolons,
// respecting string literals (single quotes).
func splitStatements(input string) []string {
	var statements []string
	var current strings.Builder
	inString := false

	for i := 0; i < len(input); i++ {
		ch := input[i]

		if ch == '\'' {
			inString = !inString
			current.WriteByte(ch)
			continue
		}

		if ch == ';' && !inString {
			stmt := strings.TrimSpace(current.String())
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	// Don't forget the last statement (may not have trailing semicolon)
	last := strings.TrimSpace(current.String())
	if last != "" {
		statements = append(statements, last)
	}

	return statements
}

// stripComments removes leading SQL comments (-- and /* */) from a statement
func stripComments(s string) string {
	s = strings.TrimSpace(s)
	for {
		if strings.HasPrefix(s, "--") {
			parts := strings.SplitN(s, "\n", 2)
			if len(parts) > 1 {
				s = strings.TrimSpace(parts[1])
				continue
			}
			return "" // entire line is a comment
		}
		if strings.HasPrefix(s, "/*") {
			end := strings.Index(s, "*/")
			if end >= 0 {
				s = strings.TrimSpace(s[end+2:])
				continue
			}
			return "" // unclosed block comment
		}
		break
	}
	return s
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
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
