// Package mcp implements the MCP (Model Context Protocol) server for Oracle database access.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alvin/oracle-mcp-server/internal/audit"
	"github.com/alvin/oracle-mcp-server/internal/config"
	"github.com/alvin/oracle-mcp-server/internal/confirm"
	"github.com/alvin/oracle-mcp-server/internal/oracle"
	"github.com/alvin/oracle-mcp-server/internal/sqlanalyzer"
)

// JSON-RPC 2.0 structures
type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP Protocol structures
type initializeParams struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    struct{}   `json:"capabilities"`
	ClientInfo      clientInfo `json:"clientInfo"`
}

type clientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type initializeResult struct {
	ProtocolVersion string           `json:"protocolVersion"`
	Capabilities    serverCapability `json:"capabilities"`
	ServerInfo      serverInfo       `json:"serverInfo"`
}

type serverCapability struct {
	Tools *toolsCapability `json:"tools,omitempty"`
}

type toolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]property `json:"properties"`
	Required   []string            `json:"required"`
}

type property struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolsListResult struct {
	Tools []tool `json:"tools"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type toolCallResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Error codes
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
	ErrCodeUserRejected   = -32000
	ErrCodeMultiStatement = -32001
	ErrCodePLSQLBlock     = -32002
	ErrCodeSQLExecution   = -32003
)

// Server is the MCP server implementation.
type Server struct {
	config       *config.Config
	executorPool *oracle.ExecutorPool
	analyzer     *sqlanalyzer.Analyzer
	confirmer    *confirm.Confirmer
	auditor      *audit.Auditor

	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex

	initialized bool
}

// NewServer creates a new MCP server.
func NewServer(cfg *config.Config) (*Server, error) {
	connections := cfg.OracleConnections()
	if connections == nil {
		return nil, fmt.Errorf("no Oracle connections in config")
	}

	executorPool, err := oracle.NewExecutorPool(connections)
	if err != nil {
		return nil, fmt.Errorf("failed to create Oracle executor pool: %w", err)
	}

	var auditor *audit.Auditor
	if cfg.Logging.AuditLog {
		logPath := cfg.Logging.LogFile
		if cfg.ConfigPath != "" && !filepath.IsAbs(logPath) {
			logPath = filepath.Join(filepath.Dir(cfg.ConfigPath), logPath)
		}
		auditor, err = audit.NewAuditor(logPath)
		if err != nil {
			executorPool.Close()
			return nil, fmt.Errorf("failed to create auditor: %w", err)
		}
	}

	return &Server{
		config:       cfg,
		executorPool: executorPool,
		analyzer:     sqlanalyzer.NewAnalyzer(cfg.Security.DangerKeywords),
		confirmer:    confirm.NewConfirmer(),
		auditor:      auditor,
		reader:       bufio.NewReader(os.Stdin),
		writer:       os.Stdout,
	}, nil
}

// Run starts the MCP server and processes requests.
func (s *Server) Run(ctx context.Context) error {
	defer s.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.processRequest(); err != nil {
				if err == io.EOF {
					return nil
				}
				// Log error but continue processing
				fmt.Fprintf(os.Stderr, "Error processing request: %v\n", err)
			}
		}
	}
}

// Close cleans up server resources.
func (s *Server) Close() {
	if s.executorPool != nil {
		s.executorPool.Close()
	}
	if s.auditor != nil {
		s.auditor.Close()
	}
}

// processRequest reads and handles a single JSON-RPC request.
func (s *Server) processRequest() error {
	line, err := s.reader.ReadBytes('\n')
	if err != nil {
		return err
	}

	var req jsonRPCRequest
	if err := json.Unmarshal(line, &req); err != nil {
		s.sendError(nil, ErrCodeParseError, "Parse error", nil)
		return nil
	}

	s.handleRequest(&req)
	return nil
}

// handleRequest routes the request to the appropriate handler.
func (s *Server) handleRequest(req *jsonRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "initialized", "notifications/initialized":
		// Notifications (no id): client signals init done, no response needed.
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(req)
	case "ping":
		s.handlePing(req)
	default:
		// Notifications have no id; do not send error response for them.
		if req.ID != nil {
			s.sendError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Method not found: %s", req.Method), nil)
		}
	}
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(req *jsonRPCRequest) {
	s.initialized = true

	result := initializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities: serverCapability{
			Tools: &toolsCapability{
				ListChanged: false,
			},
		},
		ServerInfo: serverInfo{
			Name:    "oracle-mcp-server",
			Version: "1.0.0",
		},
	}

	s.sendResult(req.ID, result)
}

// handleToolsList returns the list of available tools.
func (s *Server) handleToolsList(req *jsonRPCRequest) {
	result := toolsListResult{
		Tools: []tool{
			{
				Name:        "execute_sql",
				Description: "Execute SQL against an Oracle database. When multiple databases are configured (e.g. source and target), use the 'connection' argument to choose which one (call list_connections to see names). Supports SELECT, INSERT, UPDATE, DELETE, DDL (CREATE, DROP, ALTER, etc.), and multiple statements. Multiple statements: one per line, each line ending with a semicolon. DDL is auto-committed. SQL that matches config danger_keywords will open a confirmation window showing the full SQL.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]property{
						"sql": {
							Type:        "string",
							Description: "SQL to run: one statement, or multiple statements (one per line, each line ending with semicolon).",
						},
						"connection": {
							Type:        "string",
							Description: "Which configured database to use (e.g. 'database1', 'database2'). Required when multiple connections are configured; use list_connections to see names. Omit when only one connection is configured.",
						},
					},
					Required: []string{"sql"},
				},
			},
			{
				Name:        "list_connections",
				Description: "List the names of configured Oracle database connections. Use these names as the 'connection' argument in execute_sql when copying or syncing between databases.",
				InputSchema: inputSchema{
					Type:       "object",
					Properties: map[string]property{},
					Required:   []string{},
				},
			},
		},
	}

	s.sendResult(req.ID, result)
}

// handleToolsCall handles tool execution requests.
func (s *Server) handleToolsCall(req *jsonRPCRequest) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, ErrCodeInvalidParams, "Invalid params", nil)
		return
	}

	switch params.Name {
	case "execute_sql":
		s.handleExecuteSQL(req, params.Arguments)
	case "list_connections":
		s.handleListConnections(req)
	default:
		s.sendError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("Unknown tool: %s", params.Name), nil)
	}
}

// handleExecuteSQL handles the execute_sql tool.
func (s *Server) handleExecuteSQL(req *jsonRPCRequest, args map[string]interface{}) {
	// Extract SQL from arguments
	sqlArg, ok := args["sql"]
	if !ok {
		s.sendToolError(req.ID, "Missing required parameter: sql")
		return
	}

	sql, ok := sqlArg.(string)
	if !ok {
		s.sendToolError(req.ID, "Parameter 'sql' must be a string")
		return
	}

	// Optional: which configured connection to use (when multiple DBs are configured)
	connectionName := ""
	if c, ok := args["connection"]; ok && c != nil {
		if cs, ok := c.(string); ok {
			connectionName = strings.TrimSpace(cs)
		}
	}
	// For display/audit: when only one connection is configured, use its name instead of empty
	displayConnection := connectionName
	if displayConnection == "" {
		names := s.executorPool.Names()
		if len(names) == 1 {
			displayConnection = names[0]
		}
	}

	// Analyze the SQL
	analysis := s.analyzer.Analyze(sql)
	stmtType := sqlanalyzer.GetStatementType(sql)

	// Confirmation when SQL contains config danger_keywords or is DDL (do not match "create" inside string literals)
	needsConfirmation := analysis.IsDangerous ||
		(s.config.Security.RequireConfirmForDDL && analysis.IsDDL)

	if needsConfirmation {
		confirmReq := &confirm.ConfirmRequest{
			SQL:             sql,
			MatchedKeywords: analysis.MatchedKeywords,
			StatementType:   stmtType,
			IsDDL:           analysis.IsDDL,
			Connection:      displayConnection,
		}

		approved, err := s.confirmer.Confirm(confirmReq)
		if err != nil {
			s.logAudit(sql, analysis.MatchedKeywords, false, "CONFIRM_ERROR: "+err.Error(), displayConnection)
			s.sendToolError(req.ID, fmt.Sprintf("Confirmation dialog error: %v", err))
			return
		}

		if !approved {
			s.logAudit(sql, analysis.MatchedKeywords, false, "USER_REJECTED", displayConnection)
			s.sendError(req.ID, ErrCodeUserRejected, "Execution cancelled by user", map[string]interface{}{
				"code":             "USER_REJECTED",
				"matched_keywords": analysis.MatchedKeywords,
			})
			return
		}
	}

	// Execute the SQL on the chosen connection
	ctx := context.Background()
	result, err := s.executorPool.Execute(ctx, connectionName, sql, stmtType)
	if err != nil {
		s.logAudit(sql, analysis.MatchedKeywords, false, "EXECUTION_ERROR: "+err.Error(), displayConnection)
		s.sendToolError(req.ID, fmt.Sprintf("SQL execution failed: %v", err))
		return
	}

	// Log successful execution
	s.logAudit(sql, analysis.MatchedKeywords, true, "SUCCESS", displayConnection)

	// Format and return result
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	s.sendToolResult(req.ID, string(resultJSON))
}

// handleListConnections handles the list_connections tool.
func (s *Server) handleListConnections(req *jsonRPCRequest) {
	names := s.executorPool.Names()
	out := map[string]interface{}{
		"connections": names,
		"message":     "Use these names as the 'connection' argument in execute_sql.",
	}
	resultJSON, _ := json.MarshalIndent(out, "", "  ")
	s.sendToolResult(req.ID, string(resultJSON))
}

// handlePing handles ping requests.
func (s *Server) handlePing(req *jsonRPCRequest) {
	s.sendResult(req.ID, map[string]string{"status": "ok"})
}

// sendResult sends a successful response.
func (s *Server) sendResult(id interface{}, result interface{}) {
	s.sendResponse(&jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

// sendError sends an error response.
func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	s.sendResponse(&jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

// sendToolResult sends a successful tool result.
func (s *Server) sendToolResult(id interface{}, text string) {
	result := toolCallResult{
		Content: []contentItem{
			{Type: "text", Text: text},
		},
		IsError: false,
	}
	s.sendResult(id, result)
}

// sendToolError sends a tool error result.
func (s *Server) sendToolError(id interface{}, message string) {
	result := toolCallResult{
		Content: []contentItem{
			{Type: "text", Text: message},
		},
		IsError: true,
	}
	s.sendResult(id, result)
}

// sendResponse writes a JSON-RPC response to stdout.
func (s *Server) sendResponse(resp *jsonRPCResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return
	}

	s.writer.Write(data)
	s.writer.Write([]byte("\n"))
}

// logAudit logs an audit entry if auditing is enabled. connection is the DB alias (e.g. "database1", "database2").
func (s *Server) logAudit(sql string, keywords []string, approved bool, action string, connection string) {
	if s.auditor != nil {
		s.auditor.Log(sql, keywords, approved, action, connection)
	}
}
