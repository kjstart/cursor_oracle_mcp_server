[English](#english) | [中文](#chinese)

<a id="english"></a>

# New Project: Cursor DB MCP  
Supports connecting to any database via JDBC.  
👉[https://github.com/kjstart/cursor_db_mcp](https://github.com/kjstart/cursor_db_mcp)


# Oracle MCP Server

A Model Context Protocol (MCP) server for Oracle Database, enabling AI assistants like Cursor to execute SQL statements directly against Oracle databases.

**Alvin Liu** — [https://alvinliu.com](https://alvinliu.com) · **Project:** [https://github.com/kjstart/cursor_oracle_mcp_server](https://github.com/kjstart/cursor_oracle_mcp_server)

## 🎬 Demo Video
👉 Click the image below to watch on YouTube
[![Cursor Oracle MCP Demo](https://www.alvinliu.com/wp-content/uploads/2026/02/oracle_mcp_youtube.png)](https://www.youtube.com/watch?v=3U1nWj9tP24)

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

## Installation

### Download pre-built binary (recommended)

Pre-built binaries are published on [GitHub Releases](https://github.com/kjstart/cursor_oracle_mcp_server/releases). No build required.

1. **Download** the archive for your platform:
   - **Windows**: `oracle-mcp-server-windows-amd64-<version>.zip`
   - **macOS Apple Silicon (M1/M2/M3)**: `oracle-mcp-server-darwin-arm64-<version>.tar.gz`
   - **macOS Intel**: `oracle-mcp-server-darwin-amd64-<version>.tar.gz`

2. **Extract** the archive. You will get:
   - The executable (e.g. `oracle-mcp-server-windows-amd64.exe` or `oracle-mcp-server-darwin-arm64`)
   - `user_guide.md` — step-by-step setup
   - `config.yaml.example` — copy to `config.yaml` and edit

3. **Install Oracle Instant Client** (see [Requirements](#requirements) above) and add it to `PATH` (Windows) or set `ORACLE_HOME` and `DYLD_LIBRARY_PATH` (macOS).

4. **Configure** `config.yaml` with your database connection(s), then add the server to Cursor MCP (see [Configuration](#configuration) and [Usage with Cursor](#usage-with-cursor) below).

For a full walkthrough, see [user_guide.md](user_guide.md).

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

## Building from Source (optional)

If you prefer to build the binary yourself (e.g. for a different Go version or platform):

### Build dependencies

- Go 1.22+
- **CGO must be enabled**: godror requires CGO and a C compiler.
- **Windows**: Use **MinGW-w64** (GCC 7.2+) and add its `bin` to PATH.
  - **Do not use Cygwin's gcc.** If both are installed, ensure MinGW's `bin` comes before Cygwin in PATH, or you may see `cannot parse gcc output ... as ELF, Mach-O, PE, XCOFF`.
  - Install via Chocolatey: `choco install mingw`, or [MSYS2](https://www.msys2.org/) then `pacman -S mingw-w64-ucrt-x86_64-gcc` and use `mingw64\bin`.
  - Verify with `where gcc` and `gcc -v`; you should see "mingw" or "MinGW" and 64-bit (x86_64).
- **macOS**: Run `xcode-select --install` for command line tools.

### Build commands

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

## License

MIT License - see [LICENSE](LICENSE) file

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

## Acknowledgments

- [godror](https://github.com/godror/godror) - Oracle driver for Go
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP specification

---

<a id="chinese"></a>

[English](#english) | [中文](#chinese)

# 新项目 Cursor DB MCP 可以连接到各种数据库产品(包括OceanBase等信创数据库)
👉[https://github.com/kjstart/cursor_db_mcp](https://github.com/kjstart/cursor_db_mcp)

# Oracle MCP Server（中文）

基于 Model Context Protocol (MCP) 的 Oracle 数据库服务端，让 Cursor 等 AI 助手直接对 Oracle 数据库执行 SQL。

**作者：Alvin Liu** — [https://alvinliu.com](https://alvinliu.com) · **项目：** [https://github.com/kjstart/cursor_oracle_mcp_server](https://github.com/kjstart/cursor_oracle_mcp_server)

## 功能

- **完整 SQL 支持**：SELECT、INSERT、UPDATE、DELETE、DDL（CREATE、DROP、ALTER 等），单次请求可执行多条语句
- **从文件执行**：通过 `execute_sql_file` 执行整个 SQL 文件；自动去除末尾 SQL*Plus 的 `/`
- **PL/SQL 块**：CREATE PROCEDURE/FUNCTION/PACKAGE（含文件头部注释）及匿名块作为整体执行
- **人工确认**：可配置危险关键词，触发带完整 SQL 的确认窗口（Windows 下语法高亮）；首行：数据库 | 操作 | 关键词 | DDL，第二行：文件（来自 `execute_sql_file` 时）；焦点在 SQL 内容而非按钮
- **危险词匹配**：`whole_text`（整段 SQL 子串）或 `tokens`（精确词匹配，如 `created_at` 不匹配 `create`）
- **多数据库**：可配置多个连接；用 `list_connections` 查看名称与状态（失败连接每次列出时会重试）
- **审计日志**：键值字段（`AUDIT_TIME`、`AUDIT_CONNECTION`、`AUDIT_KEYWORDS`、`AUDIT_APPROVED`、`AUDIT_ACTION`、`AUDIT_SQL`）、完整 SQL、记录分隔符 `######AUDIT_END######`；单文件 10MB 轮转，启动时复用最近未满的日志，文件名含创建日期（如 `audit_2006-01-02_150405.log`）
- **跨平台**：Windows（WinForms + WebBrowser 确认）、macOS（osascript 对话框）
- **单可执行文件**：独立二进制（需安装 Oracle Instant Client）

## 环境要求

### 运行时依赖

1. **Oracle Instant Client**
   - 从 [Oracle 官网](https://www.oracle.com/database/technologies/instant-client/downloads.html) 下载
   - 建议 19c 或更高版本
   - 所需文件：`oci.dll`、`oraociei19.dll`（Windows）或对应 `.dylib`（macOS）

2. **环境配置**
   ```bash
   # Windows：将 Instant Client 加入 PATH
   set PATH=C:\path\to\instantclient;%PATH%
   
   # macOS：设置库路径
   export DYLD_LIBRARY_PATH=/path/to/instantclient:$DYLD_LIBRARY_PATH
   ```

## 安装

### 下载预编译包（推荐）

预编译包发布在 [GitHub Releases](https://github.com/kjstart/cursor_oracle_mcp_server/releases)，无需自行编译。

1. **下载**对应平台的压缩包：
   - **Windows**：`oracle-mcp-server-windows-amd64-<版本>.zip`
   - **macOS Apple Silicon (M1/M2/M3)**：`oracle-mcp-server-darwin-arm64-<版本>.tar.gz`
   - **macOS Intel**：`oracle-mcp-server-darwin-amd64-<版本>.tar.gz`

2. **解压**后得到：
   - 可执行文件（如 `oracle-mcp-server-windows-amd64.exe` 或 `oracle-mcp-server-darwin-arm64`）
   - `user_guide.md` — 分步设置说明
   - `config.yaml.example` — 复制为 `config.yaml` 后编辑

3. **安装 Oracle Instant Client**（见上方 [环境要求](#环境要求)），并加入 `PATH`（Windows）或设置 `ORACLE_HOME`、`DYLD_LIBRARY_PATH`（macOS）。

4. **配置** `config.yaml` 中的数据库连接，再将本服务加入 Cursor MCP（见下方 [配置](#配置) 与 [在 Cursor 中使用](#在-cursor-中使用)）。

完整步骤可参考 [user_guide.md](user_guide.md)。

## 配置

将 `config.yaml.example` 复制为 `config.yaml`，在 `oracle.connections` 下至少配置一个连接：

```yaml
oracle:
  connections:
    database1: "user/pass@//host:1521/ORCL"
    # database2: "user/pass@//host2:1521/ORCL"

security:
  # danger_keywords 匹配方式："whole_text" 或 "tokens"
  danger_keyword_match: "whole_text"

  danger_keywords:
    - truncate
    - drop
    - delete
    - create
    - update
    - execute immediate
    # ... (见 config.yaml.example)

  require_confirm_for_ddl: true   # DDL 始终需确认

logging:
  audit_log: true
  verbose_logging: true
  log_file: "audit.log"
```

**单连接**时所有 SQL 都发往该库（无需传 `connection`）。**多连接**时在 `execute_sql` / `execute_sql_file` 中通过 `connection` 指定，并用 `list_connections` 查看名称与可用性。

### 环境变量

| 变量 | 说明 |
|------|------|
| `ORACLE_MCP_CONFIG` | 配置文件路径（覆盖默认位置） |
| `ORACLE_HOME` | Oracle 客户端安装路径 |
| `PATH`（Windows） | 须包含 Instant Client 目录 |
| `TNS_ADMIN` | **连接 Oracle 自治数据库 (ADB) 时必填** — 存放 `tnsnames.ora` 及钱包文件的目录（如从 ADB 下载的 Wallet zip 解压后的目录） |

### Oracle 自治数据库 (ADB) 与 Wallet

ADB 使用 **TCPS (SSL)**，需要 **Wallet**。连接串使用钱包中 `tnsnames.ora` 的 TNS 别名（如 `mcpdemo_high`）：

1. **下载 Wallet**：Oracle Cloud 控制台 → 你的自治数据库 → **DB 连接** → **下载 Wallet**。解压到某目录（如 `D:\oracle\wallet_mcpdemo`），该目录需包含 `tnsnames.ora`、`sqlnet.ora`、`cwallet.sso`、`ewallet.pem` 等。
2. **config.yaml** — 使用 TNS 别名和数据库用户/密码：
   ```yaml
   oracle:
     connections:
       mcpdemo: "mcpdemo/YourPassword@mcpdemo_high"
   ```
3. **Cursor MCP** — 在 MCP 配置中设置 `TNS_ADMIN` 为 **Wallet 目录**，以便进程找到 `tnsnames.ora` 和 SSL 证书。Windows 下还需在 `PATH` 中包含 Instant Client：
   ```json
   {
     "mcpServers": {
       "oracle": {
         "command": "D:\\path\\to\\oracle-mcp-server-windows-amd64.exe",
         "args": [],
         "env": {
           "TNS_ADMIN": "D:\\oracle\\wallet_mcpdemo",
           "PATH": "C:\\path\\to\\instantclient;%PATH%"
         }
       }
     }
   }
   ```
   将 `D:\oracle\wallet_mcpdemo` 换成你的 Wallet 解压路径，并确保 Instant Client 在 `PATH` 中。未设置 `TNS_ADMIN` 可能出现 **ORA-12541**（无监听）或 SSL 相关错误。

## 在 Cursor 中使用

### MCP 配置

在 Cursor 的 MCP 设置（`~/.cursor/mcp.json` 或工作区 `.cursor/mcp.json`）中添加：

**Windows（Oracle 客户端已在系统 PATH 中）：**
```json
{
  "mcpServers": {
    "oracle": {
      "command": "C:\\path\\to\\oracle-mcp-server-windows-amd64.exe",
      "args": []
    }
  }
}
```

**macOS**

通过 `env` 让 MCP 进程能读到 `ORACLE_HOME` 和 `DYLD_LIBRARY_PATH`：

```json
{
  "mcpServers": {
    "oracle": {
      "command": "/path/to/oracle-mcp-server-darwin-arm64",
      "args": [],
      "env": {
        "ORACLE_HOME": "/opt/oracle/instantclient_19_20",
        "DYLD_LIBRARY_PATH": "/opt/oracle/instantclient_19_20"
      }
    }
  }
}
```

将 `/path/to/oracle-mcp-server-darwin-arm64` 和 `/opt/oracle/instantclient_19_20` 换成你的实际路径。若从已设置环境的终端启动 Cursor，也可用 `"ORACLE_HOME": "${env:ORACLE_HOME}"` 引用现有环境变量。

### 工具说明

| 工具 | 说明 |
|------|------|
| **execute_sql** | 执行 SQL（单条或多条）。参数：`sql`，可选 `connection`。 |
| **execute_sql_file** | 从文件读取 SQL，分析、必要时展示确认，再执行。末尾 `/` 会被去除。参数：`file_path`，可选 `connection`。 |
| **list_connections** | 列出已配置连接名称及可用性；会对之前失败的连接重试。 |

### 使用示例

```
// 单连接：无需传 connection
execute_sql({ "sql": "SELECT table_name FROM user_tables" })

// 多连接
execute_sql({ "sql": "SELECT * FROM my_table", "connection": "database1" })
execute_sql({ "sql": "CREATE TABLE test (id NUMBER)", "connection": "database2" })

// 执行 SQL 文件（末尾 / 会被去除）
execute_sql_file({ "file_path": "d:\\scripts\\myscript.sql", "connection": "ps" })

// 查看连接名称与状态
list_connections()
```

## 安全与确认窗口

当 SQL 命中 `danger_keywords` 或为 DDL（且 `require_confirm_for_ddl` 为 true）时，会弹出确认窗口：

- **Windows**：WinForms 窗口，SQL 语法高亮（WebBrowser）。首行：数据库 | 操作 | 关键词 | DDL；第二行：文件（来自 `execute_sql_file` 时）。焦点在 SQL 内容上。
- **macOS**：osascript 对话框显示完整 SQL。

用户确认后才会执行。拒绝会记录并返回 `USER_REJECTED`。

## SQL 执行规则

- **单条语句**：一条 SQL，可有可无末尾分号。
- **多条语句**：每行一条，**每行以分号结尾**。按顺序执行。
- **PL/SQL**：CREATE PROCEDURE/FUNCTION/PACKAGE（含文件头部 `--` 或 `/* */`）及匿名块（BEGIN...END; / DECLARE...END;）视为一整块，不拆分。
- **从文件**：单独一行的 SQL*Plus `/` 会在执行前移除。

## 审计日志

- **键值格式**：`AUDIT_TIME=...`、`AUDIT_CONNECTION=...`、`AUDIT_KEYWORDS=...`、`AUDIT_APPROVED=...`、`AUDIT_ACTION=...`、`AUDIT_SQL=` 后跟完整 SQL，再以 `######AUDIT_END######` 作为记录分隔。
- **轮转**：单文件 10MB。启动时复用最近未满的日志文件；写满后新建带创建日期的文件，如 `audit_2006-01-02_150405.log`。

## MCP 协议

### 工具：`execute_sql`

**输入**：`sql`（必填），`connection`（可选）。

**输出（查询）**：`columns`、`rows`、`statement_type`、`execution_time_ms`、`success`。

**输出（DML/DDL）**：`rows_affected`、`statement_type`、`execution_time_ms`、`success`，可选 `warning`。

**错误（用户拒绝）**：`code` -32000，`message` "Execution cancelled by user"，`data.code` "USER_REJECTED"，`data.matched_keywords`。

### 工具：`execute_sql_file`

**输入**：`file_path`（必填），`connection`（可选）。与 `execute_sql` 相同的分析与确认规则；执行文件内容（末尾 `/` 去除）。

### 工具：`list_connections`

**输入**：无。**输出**：`connections`（名称 + 可用性），`message`。

## 故障排除

### 连接问题

```
Error: ORA-12541: TNS:no listener
```
→ 确认 Oracle Instant Client 在 PATH 中且监听正常

```
Error: DPI-1047: Cannot locate a 64-bit Oracle Client library
```
→ 安装 Oracle Instant Client 并加入 PATH

### 权限问题

```
Error: ORA-01031: insufficient privileges
```
→ 当前配置的数据库用户权限不足

## 从源码构建（可选）

若希望自行编译（例如使用不同 Go 版本或平台）：

### 构建依赖

- Go 1.22+
- **须启用 CGO**：godror 依赖 CGO 和 C 编译器。
- **Windows**：使用 **MinGW-w64**（GCC 7.2+）并将其 `bin` 加入 PATH。
  - **不要用 Cygwin 的 gcc。** 若两者都有，确保 PATH 中 MinGW 的 `bin` 在 Cygwin 之前，否则可能出现 `cannot parse gcc output ... as ELF, Mach-O, PE, XCOFF`。
  - 可用 Chocolatey：`choco install mingw`，或 [MSYS2](https://www.msys2.org/) 后 `pacman -S mingw-w64-ucrt-x86_64-gcc`，使用 `mingw64\bin`。
  - 用 `where gcc` 和 `gcc -v` 检查，应看到 "mingw" 或 "MinGW" 及 64 位 (x86_64)。
- **macOS**：执行 `xcode-select --install` 安装命令行工具。

### 构建命令

**注意**：需在启用 CGO 且已安装 GCC 的环境下构建，否则会出现 `undefined: VersionInfo` 等错误。

```bash
# 克隆仓库
git clone https://github.com/kjstart/cursor_oracle_mcp_server
cd cursor_oracle_mcp_server

# 下载依赖
go mod tidy

# Windows (PowerShell)：启用 CGO，确保 gcc 在 PATH 中
$env:CGO_ENABLED="1"
go build -o oracle-mcp.exe .

# Windows (CMD)
set CGO_ENABLED=1
go build -o oracle-mcp.exe .

# macOS (Intel)
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o oracle-mcp .

# macOS (Apple Silicon)
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o oracle-mcp .
```

### 构建故障排除

| 错误 | 原因 | 处理 |
|------|------|------|
| `undefined: VersionInfo` | 未启用 CGO | 设置 `CGO_ENABLED=1` 并安装 gcc |
| `gcc not found` | 未安装 gcc 或不在 PATH | 安装 MinGW-w64 并将其 `bin` 加入 PATH |
| `cannot parse gcc output ... as ELF, Mach-O, PE, XCOFF` | 使用了错误的 gcc（如 **Cygwin gcc**） | 使用 **MinGW-w64** 的 gcc，并保证在 PATH 中位于 Cygwin 之前；用 `where gcc` 和 `gcc -v` 核对 |

修改 gcc 或 PATH 后，清理缓存再构建：

```bash
go clean -cache
go build -o oracle-mcp.exe .
```

## 许可证

MIT License - 见 [LICENSE](LICENSE) 文件

## 参与贡献

1. Fork 本仓库
2. 创建功能分支
3. 提交 Pull Request

## 致谢

- [godror](https://github.com/godror/godror) - Go 版 Oracle 驱动
- [Model Context Protocol](https://modelcontextprotocol.io/) - MCP 规范
