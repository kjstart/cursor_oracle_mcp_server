// Package oracle provides Oracle database connection and SQL execution functionality.
package oracle

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/godror/godror"

	"github.com/alvin/oracle-mcp-server/internal/sqlanalyzer"
)

// ExecutionResult contains the result of SQL execution.
type ExecutionResult struct {
	// For SELECT queries
	Columns []string        `json:"columns,omitempty"`
	Rows    [][]interface{} `json:"rows,omitempty"`

	// For DML/DDL statements
	RowsAffected int64 `json:"rows_affected,omitempty"`
	Success      bool  `json:"success"`

	// Metadata
	StatementType string `json:"statement_type"`
	ExecutionTime int64  `json:"execution_time_ms"`
	Warning       string `json:"warning,omitempty"`
}

// Executor handles Oracle database connections and SQL execution.
type Executor struct {
	db  *sql.DB
	dsn string
}

// NewExecutor creates a new Oracle executor with the given DSN.
func NewExecutor(dsn string) (*Executor, error) {
	db, err := sql.Open("godror", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &Executor{
		db:  db,
		dsn: dsn,
	}, nil
}

// Close closes the database connection.
func (e *Executor) Close() error {
	if e.db != nil {
		return e.db.Close()
	}
	return nil
}

// Execute runs the given SQL (single or multiple statements) and returns the result.
// Multiple statements are split by semicolon at end of line; single PL/SQL blocks (CREATE PROC...END;, BEGIN...END;) are not split.
func (e *Executor) Execute(ctx context.Context, sqlText string, statementType string) (*ExecutionResult, error) {
	start := time.Now()
	result := &ExecutionResult{
		StatementType: statementType,
		Success:       false,
	}

	normalized := strings.ReplaceAll(strings.TrimSpace(sqlText), "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	var statements []string
	if sqlanalyzer.IsSingleStatementBlock(normalized) {
		statements = []string{normalized}
	} else {
		statements = splitStatements(normalized)
	}

	for _, st := range statements {
		st = strings.TrimSpace(st)
		if st == "" {
			continue
		}
		if !strings.HasSuffix(st, ";") {
			st = st + ";"
		}
		// Keep trailing semicolon for PL/SQL creation (CREATE PROCEDURE/FUNCTION/PACKAGE ... END xxx;) so Oracle compiles correctly
		if !sqlanalyzer.IsPLSQLCreationStatement(st) {
			st = strings.TrimSuffix(st, ";") // Oracle driver does not want trailing semicolon for ordinary SQL
		}
		st = strings.TrimSpace(st)
		upper := strings.ToUpper(st)
		isQuery := strings.HasPrefix(upper, "SELECT") || strings.HasPrefix(upper, "WITH")
		if isQuery {
			if err := e.executeQuery(ctx, st, result); err != nil {
				return nil, err
			}
		} else {
			if err := e.executeStatement(ctx, st, result); err != nil {
				return nil, err
			}
		}
	}

	result.ExecutionTime = time.Since(start).Milliseconds()
	result.Success = true
	if isDDLStatement(statementType) {
		result.Warning = "DDL statements are auto-committed in Oracle"
	}
	return result, nil
}

// splitStatements splits SQL by semicolon at end of line (;\n). Used for multi-statement scripts.
func splitStatements(sql string) []string {
	const sep = ";\n"
	var out []string
	for {
		i := strings.Index(sql, sep)
		if i < 0 {
			break
		}
		segment := strings.TrimSpace(sql[:i])
		sql = sql[i+len(sep):]
		if segment != "" {
			out = append(out, segment)
		}
	}
	if s := strings.TrimSpace(sql); s != "" {
		out = append(out, s)
	}
	return out
}

// executeQuery handles SELECT statements.
func (e *Executor) executeQuery(ctx context.Context, sqlText string, result *ExecutionResult) error {
	rows, err := e.db.QueryContext(ctx, sqlText)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}
	result.Columns = columns

	// Prepare scan destinations
	numCols := len(columns)
	result.Rows = make([][]interface{}, 0)

	for rows.Next() {
		// Create slice to hold column values
		values := make([]interface{}, numCols)
		valuePtrs := make([]interface{}, numCols)
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert values to proper types for JSON serialization
		rowData := make([]interface{}, numCols)
		for i, v := range values {
			rowData[i] = convertValue(v)
		}
		result.Rows = append(result.Rows, rowData)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	return nil
}

// executeStatement handles DML/DDL statements.
func (e *Executor) executeStatement(ctx context.Context, sqlText string, result *ExecutionResult) error {
	execResult, err := e.db.ExecContext(ctx, sqlText)
	if err != nil {
		return fmt.Errorf("statement execution failed: %w", err)
	}

	// Get rows affected (may not be available for all statement types)
	rowsAffected, err := execResult.RowsAffected()
	if err == nil {
		result.RowsAffected = rowsAffected
	}

	return nil
}

// convertValue converts database values to JSON-serializable types.
func convertValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		return string(val)
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return val
	}
}

// isDDLStatement checks if the statement type is DDL.
func isDDLStatement(stmtType string) bool {
	ddlTypes := map[string]bool{
		"CREATE":   true,
		"DROP":     true,
		"ALTER":    true,
		"TRUNCATE": true,
		"RENAME":   true,
		"GRANT":    true,
		"REVOKE":   true,
		"COMMENT":  true,
	}
	return ddlTypes[stmtType]
}

// TestConnection tests the database connection.
func (e *Executor) TestConnection(ctx context.Context) error {
	return e.db.PingContext(ctx)
}
