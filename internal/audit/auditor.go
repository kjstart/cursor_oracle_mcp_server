// Package audit provides audit logging functionality for SQL operations.
package audit

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Auditor handles audit logging to a file.
type Auditor struct {
	file *os.File
	mu   sync.Mutex
}

// NewAuditor creates a new Auditor that writes to the specified file.
func NewAuditor(logFile string) (*Auditor, error) {
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}

	return &Auditor{
		file: file,
	}, nil
}

// Log writes an audit entry to the log file.
// connection is the database alias from config (e.g. "database1", "database2", or "default"); empty is logged as "default".
func (a *Auditor) Log(sql string, matchedKeywords []string, approved bool, action string, connection string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	timestamp := time.Now().Format(time.RFC3339)

	// Clean and truncate SQL for logging
	cleanSQL := cleanSQLForLog(sql)

	// Format keywords
	keywords := "none"
	if len(matchedKeywords) > 0 {
		keywords = strings.Join(matchedKeywords, ",")
	}

	if connection == "" {
		connection = "default"
	}

	// Write log entry (CONNECTION= alias so audit shows which DB was used)
	entry := fmt.Sprintf("%s\nCONNECTION=%s\nSQL=%s\nKEYWORDS=%s\nAPPROVED=%v\nACTION=%s\n---\n",
		timestamp, connection, cleanSQL, keywords, approved, action)

	a.file.WriteString(entry)
	a.file.Sync() // flush so audit.log is not empty when read from disk
}

// Close closes the audit log file.
func (a *Auditor) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

// cleanSQLForLog prepares SQL for logging by removing newlines and truncating if necessary.
func cleanSQLForLog(sql string) string {
	// Replace newlines with spaces
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\r", "")

	// Collapse multiple spaces
	for strings.Contains(sql, "  ") {
		sql = strings.ReplaceAll(sql, "  ", " ")
	}

	sql = strings.TrimSpace(sql)

	// Truncate if too long
	maxLen := 500
	if len(sql) > maxLen {
		sql = sql[:maxLen] + "..."
	}

	return sql
}

// AuditEntry represents a single audit log entry.
type AuditEntry struct {
	Timestamp       time.Time
	SQL             string
	MatchedKeywords []string
	Approved        bool
	Action          string
}

// Format returns a formatted string representation of the audit entry.
func (e *AuditEntry) Format() string {
	keywords := "none"
	if len(e.MatchedKeywords) > 0 {
		keywords = strings.Join(e.MatchedKeywords, ",")
	}

	return fmt.Sprintf(
		"[%s] SQL=%s | KEYWORDS=%s | APPROVED=%v | ACTION=%s",
		e.Timestamp.Format(time.RFC3339),
		cleanSQLForLog(e.SQL),
		keywords,
		e.Approved,
		e.Action,
	)
}
