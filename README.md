![Logo](https://www.alvinliu.com/wp-content/uploads/2026/02/a3f08b34-d774-48ce-453b-67e6d5cdb981.png)

# Oracle MCP Server

A Model Context Protocol (MCP) server for Oracle Database, enabling AI assistants like Cursor to execute SQL statements directly against Oracle databases.

**Alvin Liu** — [https://alvinliu.com](https://alvinliu.com) · **Project:** [https://github.com/kjstart/cursor_oracle_mcp_server](https://github.com/kjstart/cursor_oracle_mcp_server)

## Features

- **Full SQL support**: SELECT, INSERT, UPDATE, DELETE, DDL (CREATE, DROP, ALTER, etc.), and multiple statements per request
- **Execute from file**: Run a full SQL file via `execute_sql_file`; trailing SQL*Plus `/` is stripped automatically
- **PL/SQL blocks**: CREATE PROCEDURE/FUNCTION/PACKAGE (including files with leading comments) and anonymous blocks are executed as one unit
- **Human-in-the-loop**: Configurable danger keywords trigger a review window with full SQL (syntax-highlighted on Windows); Database | Action | Keywords | DDL on the first line, File on the second; focus stays on content, not buttons
- **Danger keyword matching**: `whole_text` (substring in full SQL) or `tokens` (exact token match; e.g. `created_at` does not match `create`)
- **Multi-database**: Configure multiple connections; use `list_connections` to see names and status (failed connections are retried on each list)
- **Audit logging**: Keyed fields (`AUDIT_TIME`, `AUDIT_CONNECTION`, `AUDIT_KEYWORDS`, `AUDIT_APPROVED`, `AUDIT_ACTION`, `AUDIT_SQL`), full SQL, record separator `######AUDIT_END######`; 10MB rotation, reuse last non-full file on startup, filenames include creation date (e.g. `audit_2006-01-02_150405.log`)
- **Cross-platform**: Windows (WinForms + WebBrowser for review), macOS (osascript dialog)
- **Single executable**: Standalone binary (requires Oracle Instant Client)

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
cd cursor_oracle_mcp_server

# Download dependencies
go mod tidy

# Windows (PowerShell): enable CGO and ensure gcc is in PATH
$env:CGO_ENABLED="1"
go build -o oracle-mcp.exe .

# Windows (CMD)
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
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -o oracle-mcp.exe .

# macOS (Intel): native build only
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o oracle-mcp .

# macOS (Apple Silicon)
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o oracle-mcp .
```

### Binary Distribution

```
oracle-mcp/
├── oracle-mcp.exe          # Main executable
├── config.yaml             # Configuration file (copy from config.yaml.example)
├── instantclient/          # Oracle Instant Client files
│   ├── oci.dll
│   └── oraociei19.dll
└── audit_*.log             # Audit logs (10MB rotation, creation date in filename)
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

Copy `config.yaml.example` to `config.yaml` and configure at least one connection under `oracle.connections`:

```yaml
oracle:
  connections:
    database1: "user/pass@//host:1521/ORCL"
    # database2: "user/pass@//host2:1521/ORCL"

security:
  # "whole_text" = substring in full SQL; "tokens" = exact token match (e.g. created_at ≠ create)
  danger_keyword_match: "whole_text"

  danger_keywords:
    - truncate
    - drop
    - delete
    - create
    - update
    - execute immediate
    # ... (see config.yaml.example)

  require_confirm_for_ddl: true   # DDL always requires confirmation

logging:
  audit_log: true
  verbose_logging: true   # One short stderr line per execute_sql / execute_sql_file
  log_file: "audit.log"  # Base name; actual files: audit_YYYY-MM-DD_HHMMSS.log, 10MB rotation
```

With **one** connection, all SQL runs against that database (no need to pass `connection`). With **multiple** connections, use the `connection` argument in `execute_sql` / `execute_sql_file` and `list_connections` to see names and availability.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ORACLE_MCP_CONFIG` | Path to config file (overrides default locations) |
| `ORACLE_HOME` | Oracle client installation path |
| `PATH` (Windows) | Must include Instant Client directory |
| `TNS_ADMIN` | **Required for Oracle Autonomous Database (ADB)** — directory containing `tnsnames.ora` and wallet files (e.g. `cwallet.sso`, `ewallet.pem`, `sqlnet.ora`) from the ADB Wallet zip |

### Oracle Autonomous Database (ADB) with Wallet

ADB uses **TCPS (SSL)** and requires the **Wallet**. Use the TNS name from the wallet’s `tnsnames.ora` (e.g. `mcpdemo_high`):

1. **Download Wallet**: Oracle Cloud Console → your Autonomous Database → **DB connection** → **Download Wallet**. Unzip to a folder (e.g. `D:\oracle\wallet_mcpdemo`). The folder must contain `tnsnames.ora`, `sqlnet.ora`, `cwallet.sso`, `ewallet.pem`, etc.
2. **config.yaml** — use TNS alias and your DB user/password:
   ```yaml
   oracle:
     connections:
       mcpdemo: "mcpdemo/YourPassword@mcpdemo_high"
   ```
3. **Cursor MCP** — set `TNS_ADMIN` to the **Wallet directory** so the process can find `tnsnames.ora` and SSL certs. On Windows, also include Instant Client in `PATH`:
   ```json
   {
     "mcpServers": {
       "oracle": {
         "command": "D:\\work\\code\\cursor_oracle_mcp_server\\oracle-mcp.exe",
         "args": [],
         "env": {
           "TNS_ADMIN": "D:\\oracle\\wallet_mcpdemo",
           "PATH": "C:\\path\\to\\instantclient;%PATH%"
         }
       }
     }
   }
   ```
   Replace `D:\oracle\wallet_mcpdemo` with your unzipped wallet path, and ensure Instant Client is on `PATH`. Without `TNS_ADMIN`, you may see **ORA-12541** (no listener) or SSL errors because the client cannot resolve the TNS name or use the wallet.

## Usage with Cursor

### MCP Configuration

Add to your Cursor MCP settings (`~/.cursor/mcp.json` or workspace `.cursor/mcp.json`):

**Windows:**

#### Windows, add Oracle client to PATH
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

#### Mac

Use the `env` block so the MCP process sees `ORACLE_HOME` and `DYLD_LIBRARY_PATH`:

```json
{
  "mcpServers": {
    "oracle": {
      "command": "/path/to/oracle-mcp",
      "args": [],
      "env": {
        "ORACLE_HOME": "/opt/oracle/instantclient_19_20",
        "DYLD_LIBRARY_PATH": "/opt/oracle/instantclient_19_20"
      }
    }
  }
}
```

Replace `/path/to/oracle-mcp` and `/opt/oracle/instantclient_19_20` with your actual paths. You can also reference existing shell env with `"ORACLE_HOME": "${env:ORACLE_HOME}"` if Cursor was started from a terminal that already has it set.

### Tools

| Tool | Description |
|------|-------------|
| **execute_sql** | Run SQL (one or multiple statements). Params: `sql`, optional `connection`. |
| **execute_sql_file** | Read SQL from a file, analyze, show review if needed, then execute. Trailing `/` is stripped. Params: `file_path`, optional `connection`. |
| **list_connections** | List configured connection names and availability; retries previously failed connections. |

### Example Interactions

```
// One connection: no need to pass connection
execute_sql({ "sql": "SELECT table_name FROM user_tables" })

// Multiple connections
execute_sql({ "sql": "SELECT * FROM my_table", "connection": "database1" })
execute_sql({ "sql": "CREATE TABLE test (id NUMBER)", "connection": "database2" })

// Run a SQL file (e.g. procedure script; trailing / stripped)
execute_sql_file({ "file_path": "d:\\scripts\\myscript.sql", "connection": "ps" })

// See connection names and status
list_connections()
```

## Safety and Review Window

When SQL matches `danger_keywords` or is DDL (if `require_confirm_for_ddl` is true), a confirmation window appears:

- **Windows**: WinForms window with syntax-highlighted SQL (WebBrowser). First line: Database | Action | Keywords | DDL; second line: File (when from `execute_sql_file`). Focus is on the SQL content, not the Execute/Cancel buttons.
- **macOS**: osascript dialog with full SQL.

Execution proceeds only after the user confirms. Rejection is logged and returned as `USER_REJECTED`.

## SQL Execution

- **Single statement**: One SQL statement, with or without trailing semicolon.
- **Multiple statements**: One per line, **each line ending with a semicolon**. Executed in order.
- **PL/SQL**: CREATE PROCEDURE/FUNCTION/PACKAGE (including files with leading `--` or `/* */`) and anonymous blocks (BEGIN...END; / DECLARE...END;) are treated as one block and not split.
- **From file**: Trailing SQL*Plus `/` (on its own line) is removed before execution.

## Audit Log

- **Keyed format**: `AUDIT_TIME=...`, `AUDIT_CONNECTION=...`, `AUDIT_KEYWORDS=...`, `AUDIT_APPROVED=...`, `AUDIT_ACTION=...`, `AUDIT_SQL=` followed by the full SQL, then a line `######AUDIT_END######` as record separator.
- **Rotation**: 10MB per file. On startup, the most recent existing log file under 10MB is reused; when full, a new file is created with creation date in the name: `audit_2006-01-02_150405.log`.

## MCP Protocol

### Tool: `execute_sql`

**Input**: `sql` (required), `connection` (optional).

**Output (query)**: `columns`, `rows`, `statement_type`, `execution_time_ms`, `success`.

**Output (DML/DDL)**: `rows_affected`, `statement_type`, `execution_time_ms`, `success`, optional `warning`.

**Error (user rejected)**: `code` -32000, `message` "Execution cancelled by user", `data.code` "USER_REJECTED", `data.matched_keywords`.

### Tool: `execute_sql_file`

**Input**: `file_path` (required), `connection` (optional). Same analysis and review rules as `execute_sql`; executes the file contents (trailing `/` stripped).

### Tool: `list_connections`

**Input**: none. **Output**: `connections` (name + availability), `message`.

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
