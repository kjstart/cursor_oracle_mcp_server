// Package audit provides audit logging functionality for SQL operations.
package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxAuditLogBytes = 10 << 20 // 10MB per file

// Auditor handles audit logging to a file with size-based rotation (10MB per file, filename includes creation date).
type Auditor struct {
	file        *os.File
	mu          sync.Mutex
	currentSize int64
	maxSize     int64
	dir         string
	base        string
	ext         string
}

// NewAuditor creates a new Auditor. On startup reuses the most recent existing log file that is under 10MB; only creates a new file (with creation date in name) when none exists or all are full.
func NewAuditor(logFile string) (*Auditor, error) {
	dir := filepath.Dir(logFile)
	base := strings.TrimSuffix(filepath.Base(logFile), filepath.Ext(logFile))
	if base == "" {
		base = "audit"
	}
	ext := filepath.Ext(logFile)
	if ext == "" {
		ext = ".log"
	}

	a := &Auditor{
		maxSize: maxAuditLogBytes,
		dir:     dir,
		base:    base,
		ext:     ext,
	}
	if err := a.openOrCreate(); err != nil {
		return nil, err
	}
	return a, nil
}

// openOrCreate finds the most recent existing log file under maxSize and opens it for append, or creates a new file if none.
func (a *Auditor) openOrCreate() error {
	pattern := filepath.Join(a.dir, a.base+"_*"+a.ext)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return a.rotateOpen()
	}
	// Filename base_2006-01-02_150405.log; sort descending = newest first
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Size() < a.maxSize {
			file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				continue
			}
			a.file = file
			a.currentSize = info.Size()
			return nil
		}
	}
	return a.rotateOpen()
}

// rotateOpen closes the current file (if any) and opens a new one with name base_YYYY-MM-DD_HHMMSS.ext.
func (a *Auditor) rotateOpen() error {
	if a.file != nil {
		a.file.Close()
		a.file = nil
	}
	name := fmt.Sprintf("%s_%s%s", a.base, time.Now().Format("2006-01-02_150405"), a.ext)
	path := filepath.Join(a.dir, name)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open audit log file %s: %w", path, err)
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat audit log file: %w", err)
	}
	a.file = file
	a.currentSize = info.Size()
	return nil
}

// Log writes an audit entry to the log file. When the current file reaches 10MB, a new file is opened (name includes creation date).
func (a *Auditor) Log(sql string, matchedKeywords []string, approved bool, action string, connection string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	timestamp := time.Now().Format(time.RFC3339)
	keywords := "none"
	if len(matchedKeywords) > 0 {
		keywords = strings.Join(matchedKeywords, ",")
	}
	if connection == "" {
		connection = "default"
	}

	header := fmt.Sprintf("AUDIT_TIME=%s\nAUDIT_CONNECTION=%s\nAUDIT_KEYWORDS=%s\nAUDIT_APPROVED=%v\nAUDIT_ACTION=%s\nAUDIT_SQL=\n",
		timestamp, connection, keywords, approved, action)
	entry := header + sql
	if !strings.HasSuffix(sql, "\n") {
		entry += "\n"
	}
	entry += "######AUDIT_END######\n"
	size := int64(len(entry))

	if a.currentSize+size >= a.maxSize && a.currentSize > 0 {
		if err := a.rotateOpen(); err != nil {
			// On rotate failure still write to current file to avoid losing the log entry
			_, _ = a.file.WriteString(entry)
			a.currentSize += size
			a.file.Sync()
			return
		}
	}

	a.file.WriteString(entry)
	a.currentSize += size
	a.file.Sync()
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
