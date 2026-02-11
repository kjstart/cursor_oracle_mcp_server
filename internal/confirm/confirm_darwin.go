//go:build darwin

// Package confirm provides Human-in-the-loop confirmation dialogs for macOS.
package confirm

import (
	"fmt"
	"os/exec"
	"strings"
)

// ConfirmRequest contains the data for a confirmation dialog.
type ConfirmRequest struct {
	SQL             string
	MatchedKeywords []string
	StatementType   string
	IsDDL           bool
	Connection      string // Database alias from config (e.g. "database1", "database2") for title/display
}

// Confirmer handles user confirmation dialogs on macOS.
type Confirmer struct{}

// NewConfirmer creates a new Confirmer instance.
func NewConfirmer() *Confirmer {
	return &Confirmer{}
}

// Confirm shows a confirmation dialog using osascript and returns true if the user approves.
func (c *Confirmer) Confirm(req *ConfirmRequest) (bool, error) {
	title := "Dangerous SQL Detected"
	if req.Connection != "" {
		title = "Confirm SQL — " + req.Connection
	}
	message := buildConfirmMessage(req)

	// Use osascript to display a dialog
	script := fmt.Sprintf(`
		display dialog %q with title %q buttons {"Cancel", "Execute"} default button "Cancel" with icon caution
	`, message, title)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.Output()

	if err != nil {
		// osascript returns error when user clicks Cancel
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return false, nil // User cancelled
			}
		}
		return false, fmt.Errorf("dialog error: %w", err)
	}

	// Check if user clicked Execute
	return strings.Contains(string(output), "Execute"), nil
}

// buildConfirmMessage constructs the message to display in the dialog (full SQL, no truncation).
func buildConfirmMessage(req *ConfirmRequest) string {
	var sb strings.Builder

	if req.Connection != "" {
		sb.WriteString("Database: ")
		sb.WriteString(req.Connection)
		sb.WriteString("\n\n")
	}

	// Keywords section
	if len(req.MatchedKeywords) > 0 {
		sb.WriteString("Matched Keywords: ")
		sb.WriteString(strings.ToUpper(strings.Join(req.MatchedKeywords, ", ")))
		sb.WriteString("\n\n")
	}

	// Statement type
	sb.WriteString("Statement Type: ")
	sb.WriteString(req.StatementType)
	sb.WriteString("\n\n")

	// SQL section: full SQL, no truncation
	sb.WriteString("SQL:\n")
	sb.WriteString(req.SQL)
	sb.WriteString("\n\n")

	// Warning for DDL
	if req.IsDDL {
		sb.WriteString("WARNING: Oracle DDL is auto-committed and cannot be rolled back!\n\n")
	}

	sb.WriteString("Do you want to continue?")

	return sb.String()
}

// ShowError displays an error message dialog on macOS.
func (c *Confirmer) ShowError(title, message string) {
	script := fmt.Sprintf(`
		display dialog %q with title %q buttons {"OK"} default button "OK" with icon stop
	`, message, title)
	exec.Command("osascript", "-e", script).Run()
}

// ShowInfo displays an informational message dialog on macOS.
func (c *Confirmer) ShowInfo(title, message string) {
	script := fmt.Sprintf(`
		display dialog %q with title %q buttons {"OK"} default button "OK" with icon note
	`, message, title)
	exec.Command("osascript", "-e", script).Run()
}

// Available returns true on macOS.
func (c *Confirmer) Available() bool {
	return true
}

// PlatformName returns the platform name.
func (c *Confirmer) PlatformName() string {
	return "darwin"
}

// FormatConfirmationMessage formats the confirmation message for logging.
func FormatConfirmationMessage(req *ConfirmRequest) string {
	conn := req.Connection
	if conn == "" {
		conn = "default"
	}
	return fmt.Sprintf(
		"Connection=[%s] SQL=[%s] Keywords=[%s] Type=[%s] IsDDL=[%v]",
		conn,
		truncateSQL(req.SQL, 100),
		strings.Join(req.MatchedKeywords, ","),
		req.StatementType,
		req.IsDDL,
	)
}

func truncateSQL(sql string, maxLen int) string {
	sql = strings.ReplaceAll(sql, "\n", " ")
	sql = strings.ReplaceAll(sql, "\r", "")
	if len(sql) > maxLen {
		return sql[:maxLen] + "..."
	}
	return sql
}
