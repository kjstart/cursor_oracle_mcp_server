// Package sqlanalyzer provides SQL safety analysis functionality.
// It handles comment/string removal and keyword matching for dangerous SQL detection.
package sqlanalyzer

import (
	"regexp"
	"strings"
	"unicode"
)

// AnalysisResult contains the result of SQL analysis.
type AnalysisResult struct {
	// OriginalSQL is the input SQL statement.
	OriginalSQL string
	// NormalizedSQL is the SQL after removing comments and string literals.
	NormalizedSQL string
	// Tokens are the extracted tokens from the normalized SQL.
	Tokens []string
	// MatchedKeywords contains the dangerous keywords found in the SQL.
	MatchedKeywords []string
	// IsDangerous indicates if the SQL contains any dangerous keywords.
	IsDangerous bool
	// IsDDL indicates if the SQL is a DDL statement.
	IsDDL bool
	// IsMultiStatement indicates if the SQL contains multiple statements.
	IsMultiStatement bool
	// ContainsPLSQL indicates if the SQL contains PL/SQL blocks.
	ContainsPLSQL bool
	// IsPLSQLCreationDDL is true when the SQL is a single CREATE PROCEDURE/FUNCTION/PACKAGE ... END; (allowed to run).
	IsPLSQLCreationDDL bool
}

// Analyzer performs SQL safety analysis.
type Analyzer struct {
	dangerKeywords []string
	ddlKeywords    []string
	matchMode      string // "whole_text" or "tokens"
}

// NewAnalyzer creates a new SQL analyzer with the given danger keywords and match mode.
// matchMode: "whole_text" = case-insensitive substring match on full SQL (default, stricter);
// "tokens" = match on tokens after removing comments/string literals (fewer false positives).
func NewAnalyzer(dangerKeywords []string, matchMode string) *Analyzer {
	// Normalize all keywords to lowercase
	normalized := make([]string, len(dangerKeywords))
	for i, kw := range dangerKeywords {
		normalized[i] = strings.ToLower(strings.TrimSpace(kw))
	}
	mode := strings.ToLower(strings.TrimSpace(matchMode))
	if mode != "whole_text" && mode != "tokens" {
		mode = "whole_text"
	}

	return &Analyzer{
		dangerKeywords: normalized,
		ddlKeywords: []string{
			"create",
			"drop",
			"alter",
			"truncate",
			"rename",
			"comment",
			"grant",
			"revoke",
		},
		matchMode: mode,
	}
}

// Analyze performs comprehensive SQL analysis.
func (a *Analyzer) Analyze(sql string) *AnalysisResult {
	result := &AnalysisResult{
		OriginalSQL: sql,
	}

	// Step 1: Remove comments
	noComments := removeComments(sql)

	// Step 2: Remove string literals
	noStrings := removeStringLiterals(noComments)

	result.NormalizedSQL = noStrings

	// Step 3: Check for multiple statements and PL/SQL creation DDL
	result.IsPLSQLCreationDDL = isPLSQLCreationDDL(noStrings)
	result.IsMultiStatement = isMultiStatement(noStrings)

	// Step 4: Check for PL/SQL blocks (unless it's a CREATE PROCEDURE/FUNCTION/PACKAGE)
	result.ContainsPLSQL = !result.IsPLSQLCreationDDL && containsPLSQL(noStrings)

	// Step 5: Tokenize
	result.Tokens = tokenize(noStrings)

	// Step 6: Check for DDL
	result.IsDDL = a.isDDL(result.Tokens)

	// Step 7: Match danger keywords (by full SQL or by tokens depending on mode)
	if a.matchMode == "whole_text" {
		result.MatchedKeywords = a.matchKeywordsWholeText(sql)
	} else {
		result.MatchedKeywords = a.matchKeywords(result.Tokens)
	}
	result.IsDangerous = len(result.MatchedKeywords) > 0

	return result
}

// removeComments removes SQL comments (-- and /* */).
func removeComments(sql string) string {
	// Remove single-line comments (-- comment)
	singleLinePattern := regexp.MustCompile(`--[^\r\n]*`)
	sql = singleLinePattern.ReplaceAllString(sql, " ")

	// Remove multi-line comments (/* comment */)
	multiLinePattern := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	sql = multiLinePattern.ReplaceAllString(sql, " ")

	return sql
}

// removeStringLiterals removes string literals ('string') to prevent false positives.
// Example: SELECT 'drop table' FROM dual; should not match "drop table"
func removeStringLiterals(sql string) string {
	var result strings.Builder
	inString := false
	prevChar := rune(0)

	for _, char := range sql {
		if char == '\'' && prevChar != '\'' {
			if inString {
				// Check for escaped quote ('')
				inString = false
			} else {
				inString = true
			}
			result.WriteRune(' ')
		} else if inString {
			// Skip characters inside string literals
			result.WriteRune(' ')
		} else {
			result.WriteRune(char)
		}
		prevChar = char
	}

	return result.String()
}

// IsSingleStatementBlock reports whether the entire SQL should be executed as one (no split).
// True for CREATE PROCEDURE/FUNCTION/PACKAGE ... END; or BEGIN...END; / DECLARE...END;
func IsSingleStatementBlock(sql string) bool {
	return isPLSQLCreationDDL(sql) || isAnonymousBlock(sql)
}

func isAnonymousBlock(sql string) bool {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return false
	}
	if strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSuffix(trimmed, ";")
	}
	lower := strings.ToLower(trimmed)
	hasEnd := strings.Contains(lower, " end ") || strings.HasSuffix(lower, " end")
	return (strings.HasPrefix(lower, "begin") || strings.HasPrefix(lower, "declare")) && hasEnd
}

// IsPLSQLCreationStatement reports whether the SQL is a CREATE PROCEDURE/FUNCTION/PACKAGE ... END; block.
// When true, the executor should not strip the trailing semicolon (Oracle requires it for PL/SQL compilation).
func IsPLSQLCreationStatement(sql string) bool {
	return isPLSQLCreationDDL(sql)
}

// KeepTrailingSemicolon reports whether the statement should be sent to Oracle with its trailing semicolon.
// True for CREATE PROC/FUNC/PACKAGE and for anonymous blocks (BEGIN...END;), which require the final semicolon.
func KeepTrailingSemicolon(sql string) bool {
	return isPLSQLCreationDDL(sql) || isAnonymousBlock(sql)
}

// isPLSQLCreationDDL reports whether the SQL is a single CREATE PROCEDURE/FUNCTION/PACKAGE ... END; block.
// Leading comments (-- or /* */) and blank lines are ignored so that files starting with comments are still detected.
func isPLSQLCreationDDL(sql string) bool {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	// Skip BOM
	if strings.HasPrefix(lower, "\ufeff") {
		lower = lower[1:]
	}
	// Find first "create" (start of statement after leading comments)
	idx := strings.Index(lower, "create")
	if idx == -1 {
		return false
	}
	// From first "create" onward, must look like CREATE [OR REPLACE] PROCEDURE/FUNCTION/PACKAGE ... END
	stmt := lower[idx:]
	if !strings.HasPrefix(stmt, "create") {
		return false
	}
	hasPlsql := strings.Contains(stmt, "procedure") || strings.Contains(stmt, " function ") || strings.Contains(stmt, " package ")
	hasEnd := strings.Contains(stmt, " end ") ||
		strings.Contains(stmt, " end;") ||
		strings.Contains(stmt, "\nend ") ||
		strings.Contains(stmt, "\nend;") ||
		strings.HasSuffix(stmt, " end") ||
		strings.HasSuffix(stmt, " end;")
	return hasPlsql && hasEnd
}

// isMultiStatement checks if the SQL contains multiple statements.
func isMultiStatement(sql string) bool {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return false
	}
	if strings.HasSuffix(trimmed, ";") {
		trimmed = strings.TrimSuffix(trimmed, ";")
	}
	if !strings.Contains(trimmed, ";") {
		return false
	}
	// Single CREATE PROCEDURE/FUNCTION/PACKAGE ... END; is one statement (PL/SQL body has semicolons)
	if isPLSQLCreationDDL(sql) {
		return false
	}
	// Anonymous PL/SQL block BEGIN ... END; or DECLARE ... BEGIN ... END; is one statement
	lower := strings.ToLower(trimmed)
	hasEnd := strings.Contains(lower, " end ") || strings.HasSuffix(lower, " end")
	if (strings.HasPrefix(lower, "begin") || strings.HasPrefix(lower, "declare")) && hasEnd {
		return false
	}
	return true
}

// containsPLSQL checks if the SQL contains PL/SQL blocks.
func containsPLSQL(sql string) bool {
	lower := strings.ToLower(sql)
	tokens := tokenize(lower)

	// Check for anonymous blocks
	for i, token := range tokens {
		if token == "begin" {
			// Check if there's a matching END
			for j := i + 1; j < len(tokens); j++ {
				if tokens[j] == "end" {
					return true
				}
			}
		}
		if token == "declare" {
			return true
		}
	}

	return false
}

// tokenize splits the SQL into tokens.
func tokenize(sql string) []string {
	// Convert to lowercase
	lower := strings.ToLower(sql)

	// Split by non-alphanumeric characters (except underscore)
	var tokens []string
	var currentToken strings.Builder

	for _, char := range lower {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' {
			currentToken.WriteRune(char)
		} else {
			if currentToken.Len() > 0 {
				tokens = append(tokens, currentToken.String())
				currentToken.Reset()
			}
		}
	}

	// Don't forget the last token
	if currentToken.Len() > 0 {
		tokens = append(tokens, currentToken.String())
	}

	return tokens
}

// isDDL checks if the SQL is a DDL statement.
func (a *Analyzer) isDDL(tokens []string) bool {
	if len(tokens) == 0 {
		return false
	}

	firstToken := tokens[0]
	for _, ddlKw := range a.ddlKeywords {
		if firstToken == ddlKw {
			return true
		}
	}

	return false
}

// matchKeywordsWholeText finds all danger keywords as case-insensitive substrings in the full SQL.
// Any occurrence (in string literals, comments, object names, etc.) triggers a match.
func (a *Analyzer) matchKeywordsWholeText(sql string) []string {
	lower := strings.ToLower(sql)
	var matched []string
	for _, kw := range a.dangerKeywords {
		if strings.Contains(lower, kw) {
			matched = append(matched, kw)
		}
	}
	return matched
}

// matchKeywords finds all danger keywords in the tokens.
func (a *Analyzer) matchKeywords(tokens []string) []string {
	var matched []string
	seen := make(map[string]bool)

	for _, kw := range a.dangerKeywords {
		if seen[kw] {
			continue
		}

		// Handle multi-word keywords (e.g., "alter system", "grant dba")
		kwTokens := tokenize(kw)
		if len(kwTokens) == 0 {
			continue
		}

		if len(kwTokens) == 1 {
			// Single-word keyword - exact token match
			for _, token := range tokens {
				if token == kwTokens[0] {
					matched = append(matched, kw)
					seen[kw] = true
					break
				}
			}
		} else {
			// Multi-word keyword - consecutive token match
			if matchConsecutiveTokens(tokens, kwTokens) {
				matched = append(matched, kw)
				seen[kw] = true
			}
		}
	}

	return matched
}

// matchConsecutiveTokens checks if kwTokens appear consecutively in tokens.
func matchConsecutiveTokens(tokens, kwTokens []string) bool {
	if len(kwTokens) > len(tokens) {
		return false
	}

	for i := 0; i <= len(tokens)-len(kwTokens); i++ {
		match := true
		for j, kwToken := range kwTokens {
			if tokens[i+j] != kwToken {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}

	return false
}

// GetStatementType returns the type of SQL statement.
func GetStatementType(sql string) string {
	noComments := removeComments(sql)
	noStrings := removeStringLiterals(noComments)
	tokens := tokenize(noStrings)

	if len(tokens) == 0 {
		return "UNKNOWN"
	}

	switch tokens[0] {
	case "select":
		return "SELECT"
	case "insert":
		return "INSERT"
	case "update":
		return "UPDATE"
	case "delete":
		return "DELETE"
	case "create":
		return "CREATE"
	case "drop":
		return "DROP"
	case "alter":
		return "ALTER"
	case "truncate":
		return "TRUNCATE"
	case "grant":
		return "GRANT"
	case "revoke":
		return "REVOKE"
	case "rename":
		return "RENAME"
	case "comment":
		return "COMMENT"
	default:
		return strings.ToUpper(tokens[0])
	}
}
