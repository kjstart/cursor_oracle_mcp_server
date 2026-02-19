[English](README.md) | [中文](README.zh-CN.md)

![Logo](https://www.alvinliu.com/wp-content/uploads/2026/02/a3f08b34-d774-48ce-453b-67e6d5cdb981.png)

# Oracle MCP Server

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
