package sqlanalyzer

import (
	"testing"
)

func TestRemoveComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single line comment",
			input:    "SELECT * FROM users -- this is a comment",
			expected: "SELECT * FROM users  ",
		},
		{
			name:     "multi line comment",
			input:    "SELECT /* comment */ * FROM users",
			expected: "SELECT   * FROM users",
		},
		{
			name:     "no comments",
			input:    "SELECT * FROM users",
			expected: "SELECT * FROM users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeComments(tt.input)
			if result != tt.expected {
				t.Errorf("removeComments(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRemoveStringLiterals(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "string literal with drop",
			input:    "SELECT 'drop table' FROM dual",
			contains: "drop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := removeStringLiterals(tt.input)
			tokens := tokenize(result)
			for _, token := range tokens {
				if token == tt.contains {
					t.Errorf("removeStringLiterals should have removed %q from %q", tt.contains, tt.input)
				}
			}
		})
	}
}

func TestAnalyzer_Analyze(t *testing.T) {
	analyzer := NewAnalyzer([]string{"truncate", "drop", "alter system"})

	tests := []struct {
		name             string
		sql              string
		wantDangerous    bool
		wantDDL          bool
		wantKeywords     []string
		wantMultiStmt    bool
		wantPLSQL        bool
	}{
		{
			name:          "simple select",
			sql:           "SELECT * FROM users",
			wantDangerous: false,
			wantDDL:       false,
			wantKeywords:  nil,
		},
		{
			name:          "truncate table",
			sql:           "TRUNCATE TABLE users",
			wantDangerous: true,
			wantDDL:       true,
			wantKeywords:  []string{"truncate"},
		},
		{
			name:          "drop table",
			sql:           "DROP TABLE users",
			wantDangerous: true,
			wantDDL:       true,
			wantKeywords:  []string{"drop"},
		},
		{
			name:          "alter system",
			sql:           "ALTER SYSTEM SET some_param = 'value'",
			wantDangerous: true,
			wantDDL:       true,
			wantKeywords:  []string{"alter system"},
		},
		{
			name:          "string literal with drop - should not match",
			sql:           "SELECT 'drop table' FROM dual",
			wantDangerous: false,
			wantDDL:       false,
			wantKeywords:  nil,
		},
		{
			name:          "comment with drop - should not match",
			sql:           "SELECT * FROM users -- drop table",
			wantDangerous: false,
			wantDDL:       false,
			wantKeywords:  nil,
		},
		{
			name:          "multi statement",
			sql:           "SELECT 1 FROM dual; SELECT 2 FROM dual",
			wantDangerous: false,
			wantMultiStmt: true,
		},
		{
			name:          "plsql block",
			sql:           "BEGIN NULL; END;",
			wantDangerous: false,
			wantPLSQL:     true,
		},
		{
			name:          "create table - DDL but not dangerous",
			sql:           "CREATE TABLE test (id NUMBER)",
			wantDangerous: false,
			wantDDL:       true,
			wantKeywords:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.Analyze(tt.sql)

			if result.IsDangerous != tt.wantDangerous {
				t.Errorf("IsDangerous = %v, want %v", result.IsDangerous, tt.wantDangerous)
			}
			if result.IsDDL != tt.wantDDL {
				t.Errorf("IsDDL = %v, want %v", result.IsDDL, tt.wantDDL)
			}
			if result.IsMultiStatement != tt.wantMultiStmt {
				t.Errorf("IsMultiStatement = %v, want %v", result.IsMultiStatement, tt.wantMultiStmt)
			}
			if result.ContainsPLSQL != tt.wantPLSQL {
				t.Errorf("ContainsPLSQL = %v, want %v", result.ContainsPLSQL, tt.wantPLSQL)
			}

			if tt.wantKeywords != nil {
				if len(result.MatchedKeywords) != len(tt.wantKeywords) {
					t.Errorf("MatchedKeywords = %v, want %v", result.MatchedKeywords, tt.wantKeywords)
				} else {
					for i, kw := range tt.wantKeywords {
						if result.MatchedKeywords[i] != kw {
							t.Errorf("MatchedKeywords[%d] = %v, want %v", i, result.MatchedKeywords[i], kw)
						}
					}
				}
			}
		})
	}
}

func TestGetStatementType(t *testing.T) {
	tests := []struct {
		sql      string
		expected string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"INSERT INTO users VALUES (1)", "INSERT"},
		{"UPDATE users SET name = 'test'", "UPDATE"},
		{"DELETE FROM users", "DELETE"},
		{"CREATE TABLE test (id NUMBER)", "CREATE"},
		{"DROP TABLE test", "DROP"},
		{"TRUNCATE TABLE test", "TRUNCATE"},
		{"ALTER TABLE test ADD col NUMBER", "ALTER"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := GetStatementType(tt.sql)
			if result != tt.expected {
				t.Errorf("GetStatementType(%q) = %q, want %q", tt.sql, result, tt.expected)
			}
		})
	}
}
