// Package config handles configuration loading and validation for oracle-mcp-server.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure.
type Config struct {
	Oracle   OracleConfig   `yaml:"oracle"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`

	// ConfigPath is the path to the loaded config file (set by Load); used to resolve relative paths like audit log.
	ConfigPath string `yaml:"-"`
}

// OracleConfig holds Oracle database connection settings.
// Connections: name -> DSN. Names are used as the "connection" argument in execute_sql.
// If only one connection is configured, it is used for all SQL (connection argument optional).
type OracleConfig struct {
	Connections map[string]string `yaml:"connections"`
}

// SecurityConfig holds security-related settings.
type SecurityConfig struct {
	DangerKeywords      []string `yaml:"danger_keywords"`
	DangerKeywordMatch  string   `yaml:"danger_keyword_match"` // "whole_text" (default) or "tokens"
	RequireConfirmForDDL bool    `yaml:"require_confirm_for_ddl"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	AuditLog       bool   `yaml:"audit_log"`
	VerboseLogging bool   `yaml:"verbose_logging"` // when true, log one line per execute_sql: [debug] Execute Action: <type>, Connection: <name>
	LogFile        string `yaml:"log_file"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Oracle: OracleConfig{
			Connections: nil,
		},
		Security: SecurityConfig{
			DangerKeywords: []string{
				"truncate",
				"drop",
				"alter system",
				"shutdown",
				"grant dba",
				"delete",
			},
			DangerKeywordMatch:  "whole_text",
			RequireConfirmForDDL: true,
		},
		Logging: LoggingConfig{
			AuditLog:       true,
			VerboseLogging: true,
			LogFile:        "audit.log",
		},
	}
}

// Load reads and parses the configuration file.
// It looks for the config file in the following order:
// 1. Path specified by ORACLE_MCP_CONFIG environment variable
// 2. config.yaml in the executable's directory
// 3. config.yaml in the current working directory
func Load() (*Config, error) {
	configPath := findConfigPath()
	if configPath == "" {
		return nil, fmt.Errorf("config file not found: please create config.yaml or set ORACLE_MCP_CONFIG")
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		return nil, err
	}
	cfg.ConfigPath = configPath
	return cfg, nil
}

// LoadFromFile reads and parses a configuration file from the specified path.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Normalize danger keywords to lowercase
	for i, kw := range config.Security.DangerKeywords {
		config.Security.DangerKeywords[i] = strings.ToLower(strings.TrimSpace(kw))
	}
	// Default danger keyword match mode (before Validate)
	if config.Security.DangerKeywordMatch == "" {
		config.Security.DangerKeywordMatch = "whole_text"
	} else {
		config.Security.DangerKeywordMatch = strings.ToLower(strings.TrimSpace(config.Security.DangerKeywordMatch))
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if len(c.Oracle.Connections) == 0 {
		return fmt.Errorf("oracle.connections is required and must have at least one entry")
	}
	mode := c.Security.DangerKeywordMatch
	if mode != "whole_text" && mode != "tokens" {
		return fmt.Errorf("security.danger_keyword_match must be \"whole_text\" or \"tokens\", got %q", mode)
	}
	return nil
}

// OracleConnections returns the configured connection map (name -> DSN).
func (c *Config) OracleConnections() map[string]string {
	return c.Oracle.Connections
}

// findConfigPath searches for the configuration file in standard locations.
func findConfigPath() string {
	// 1. Check environment variable
	if envPath := os.Getenv("ORACLE_MCP_CONFIG"); envPath != "" {
		if fileExists(envPath) {
			return envPath
		}
	}

	// 2. Check executable directory
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		configPath := filepath.Join(exeDir, "config.yaml")
		if fileExists(configPath) {
			return configPath
		}
	}

	// 3. Check current working directory
	if cwd, err := os.Getwd(); err == nil {
		configPath := filepath.Join(cwd, "config.yaml")
		if fileExists(configPath) {
			return configPath
		}
	}

	return ""
}

// fileExists checks if a file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
