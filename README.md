![Logo](https://www.alvinliu.com/wp-content/uploads/2026/02/a3f08b34-d774-48ce-453b-67e6d5cdb981.png)

# Oracle MCP Server

A Model Context Protocol (MCP) server for Oracle Database, enabling AI assistants like Cursor to execute SQL statements directly against Oracle databases.

## Features

- **Full SQL Support**: Execute SELECT, INSERT, UPDATE, DELETE, and DDL statements
- **Human-in-the-Loop**: Configurable safety confirmations for dangerous operations
- **Cross-Platform**: Supports Windows and macOS
- **Single Executable**: Standalone binary (requires Oracle Instant Client)
- **Audit Logging**: Complete audit trail of all SQL executions

## Requirements

### Runtime Dependencies

1. **Oracle Instant Client**
   - Download from [Oracle website](https://www.oracle.com/database/technologies/instant-client/downloads.html)
   - Version 19c or later recommended
   - Required files: `oci.dll`, `oraociei19.dll` (Windows) or equivalent `.dylib` (macOS)

2. **Environment Setup**
   ```bash
   # Windows: Add Instant Client to PATH
   set PATH=C:\path\to\instantclient;%PATH%
   
   # macOS: Set library path
   export DYLD_LIBRARY_PATH=/path/to/instantclient:$DYLD_LIBRARY_PATH
   ```

### Build Dependencies

- Go 1.22+
- **CGO must be enabled**: godror requires CGO and a C compiler.
- **Windows**: Use **MinGW-w64** (GCC 7.2+) and add its `bin` to PATH.
  - **Do not use Cygwin's gcc.** If both are installed, ensure MinGW's `bin` comes before Cygwin in PATH, or you may see `cannot parse gcc output ... as ELF, Mach-O, PE, XCOFF`.
  - Install via Chocolatey: `choco install mingw`, or [MSYS2](https://www.msys2.org/) then `pacman -S mingw-w64-ucrt-x86_64-gcc` and use `mingw64\bin`.
  - Verify with `where gcc` and `gcc -v`; you should see "mingw" or "MinGW" and 64-bit (x86_64).
- **macOS**: Run `xcode-select --install` for command line tools.

## Installation

### Building from Source

**Important**: Build with CGO enabled and GCC available, or you will see errors like `undefined: VersionInfo`.

```bash
# Clone the repository
git clone https://github.com/kjstart/cursor_oracle_mcp_server
cd oracle-mcp-server

# Download dependencies
go mod tidy

# Windows (PowerShell): enable CGO and ensure gcc is in PATH
$env:CGO_ENABLED="1"
go build -o oracle-mcp.exe .

# Windows（CMD）
set CGO_ENABLED=1
go build -o oracle-mcp.exe .

# Windows (MSYS UCRT64)
export PATH=/c/oracle/instantclient_21_12:$PATH
export ORACLE_HOME=/c/oracle/instantclient_21_12
pacman -S mingw-w64-ucrt-x86_64-go
export GOROOT=/ucrt64/lib/go
export PATH=$GOROOT/bin:$PATH
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc
go install github.com/godror/godror@latest
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o oracle-mcp.exe

# macOS (Intel): native build only
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o oracle-mcp .

# macOS (Apple Silicon)
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o oracle-mcp .
```

### Binary Distribution

```
oracle-mcp/
├── oracle-mcp.exe          # Main executable
├── config.yaml             # Configuration file
├── instantclient/          # Oracle Instant Client files
│   ├── oci.dll
│   └── oraociei19.dll
└── audit.log               # Audit log (created on first run)
```

### Build troubleshooting

| Error | Cause | Fix |
|-------|------|-----|
| `undefined: VersionInfo` | CGO disabled | Set `CGO_ENABLED=1` and install gcc |
| `gcc not found` | gcc not installed or not in PATH | Install MinGW-w64 and add its `bin` to PATH |
| `cannot parse gcc output ... as ELF, Mach-O, PE, XCOFF` | Wrong gcc (e.g. **Cygwin gcc**) | Use **MinGW-w64** gcc and put it before Cygwin in PATH; verify with `where gcc` and `gcc -v` |

After changing gcc or PATH, clear the build cache and rebuild:

```bash
go clean -cache
go build -o oracle-mcp.exe .
```

## Configuration

Copy `config.yaml.example` to `config.yaml` and configure `oracle.connections` (at least one entry):

```yaml
oracle:
  connections:
    database1: "user/pass@//localhost:1521/ORCL"
    # Add more for multi-DB (e.g. copy between servers). Use list_connections / execute_sql(connection: "name").

# Security Settings
security:
  danger_keywords:
    - truncate
    - drop
    - alter system
    - shutdown
    - grant dba
    - delete
  require_confirm_for_ddl: true

# Logging Settings
logging:
  audit_log: true
  log_file: "audit.log"
```

With **one** connection, all SQL runs against that database (no need to pass `connection`). With **multiple** connections, use the `connection` argument in `execute_sql` and `list_connections` to see names.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ORACLE_MCP_CONFIG` | Path to config file (overrides default locations) |
| `ORACLE_HOME` | Oracle client installation path |
| `PATH` (Windows) | Must include Instant Client directory |

## Usage with Cursor

### MCP Configuration

Add to your Cursor MCP settings (`~/.cursor/mcp.json` or workspace settings):

```json
{
  "mcpServers": {
    "oracle": {
      "command": "C:\\path\\to\\oracle-mcp.exe",
      "args": []
    }
  }
}
```

### Example Interactions

Tools: **execute_sql**, **list_connections**.

```
// One connection configured: no need to pass connection
execute_sql({ "sql": "SELECT table_name FROM user_tables" })

// Multiple connections: specify which DB
execute_sql({ "sql": "SELECT * FROM my_table", "connection": "database1" })
execute_sql({ "sql": "CREATE TABLE ...", "connection": "database2" })

// See configured connection names
list_connections()

// DML / DDL (will prompt for confirmation when dangerous)
execute_sql({ "sql": "CREATE TABLE test (id NUMBER PRIMARY KEY)" })
```

## Safety Features

### Human-in-the-Loop

When SQL contains dangerous keywords or is a DDL statement, a native dialog appears:

**Windows**: Win32 MessageBox  
**macOS**: osascript dialog

The dialog shows:
- Matched keywords
- Statement type
- Full SQL (truncated if long)
- DDL auto-commit warning

User must click "Yes" to proceed or "No" to cancel.

### SQL execution

- **Single statement**: Pass one SQL statement (with or without a trailing semicolon).
- **Multiple statements**: One statement per line, **each line ending with a semicolon**, e.g.:
  ```sql
  ALTER TABLE t ADD col1 NUMBER;
  INSERT INTO t VALUES (1);
  COMMIT;
  ```
  Statements are executed in order. Procedures and anonymous blocks (`BEGIN...END;`) are treated as one unit and are not split.

### Security

- **Danger keywords**: When SQL matches `danger_keywords` in config (or is DDL / contains `create`), a confirmation window shows the full SQL; execution proceeds only after the user confirms.
- **Token matching**: Only real keywords are matched; text inside string literals is ignored (e.g. `SELECT 'drop table' FROM dual` is not treated as dangerous).

## MCP Protocol

### Tool: `execute_sql`

**Input Schema**:
```json
{
  "type": "object",
  "properties": {
    "sql": {
      "type": "string",
      "description": "SQL to run: one statement, or multiple statements (one per line, each line ending with semicolon)"
    }
  },
  "required": ["sql"]
}
```

**Output (Query)**:
```json
{
  "columns": ["TABLE_NAME", "NUM_ROWS"],
  "rows": [
    ["EMPLOYEES", 107],
    ["DEPARTMENTS", 27]
  ],
  "statement_type": "SELECT",
  "execution_time_ms": 45,
  "success": true
}
```

**Output (DML/DDL)**:
```json
{
  "rows_affected": 1,
  "statement_type": "INSERT",
  "execution_time_ms": 12,
  "success": true,
  "warning": "DDL statements are auto-committed in Oracle"
}
```

**Error (User Rejected)**:
```json
{
  "error": {
    "code": -32000,
    "message": "Execution cancelled by user",
    "data": {
      "code": "USER_REJECTED",
      "matched_keywords": ["truncate"]
    }
  }
}
```

## Audit Log Format

```
2026-02-10T15:41:12+08:00
SQL=TRUNCATE TABLE user_log
KEYWORDS=truncate
APPROVED=true
ACTION=SUCCESS
---
```

## Troubleshooting

### Connection Issues

```
Error: ORA-12541: TNS:no listener
```
→ Check that Oracle Instant Client is in PATH and listener is running

```
Error: DPI-1047: Cannot locate a 64-bit Oracle Client library
```
→ Install Oracle Instant Client and add to PATH

### Permission Issues

```
Error: ORA-01031: insufficient privileges
```
→ The configured database user lacks required permissions

## License

MIT License - see [LICENSE](LICENSE) file

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## Acknowledgments

- [godror](https://github.com/godror/godror) - Oracle driver for Go
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP specification
