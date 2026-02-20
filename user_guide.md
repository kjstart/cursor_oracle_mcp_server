[English](#english) | [中文](#chinese)

<a id="english"></a>

# Oracle MCP Server — User Guide

**Author:** [Alvin Liu ](https://alvinliu.com)

**GitHub:** [https://github.com/kjstart/cursor_oracle_mcp_server](https://github.com/kjstart/cursor_oracle_mcp_server)

This guide assumes you have downloaded the release zip containing the binary for your platform (e.g. `oracle-mcp-server-windows-amd64.exe`, `oracle-mcp-server-darwin-arm64`, or `oracle-mcp-server-darwin-amd64`) and `config.yaml.example`, and extracted it locally.

---

## 1. Download and configure Oracle Client

1. **Download Oracle Instant Client**
   - Go to: [Oracle Instant Client downloads](https://www.oracle.com/database/technologies/instant-client/downloads.html)
   - Choose the package that matches your system (Windows 64-bit, macOS Intel/ARM) and download the **Basic** package
   - Version 19c or later is recommended
   - Required files: `oci.dll`, `oraociei19.dll` (Windows) or equivalent `.dylib` (macOS)

2. **Extract and set environment**

   **Windows**
   - Extract the zip to a folder, e.g. `C:\oracle\instantclient_19_20`
   - Add that folder to your system **PATH**:
     - Right-click **This PC** → **Properties** → **Advanced system settings** → **Environment variables**
     - Under **System variables**, select `Path`, click **Edit**, then **New** and add: `C:\oracle\instantclient_19_20` (use your actual path)
   - **Restart** your terminal or Cursor so the new PATH takes effect

   **macOS**
   - Extract the zip to a folder, e.g. `/opt/oracle/instantclient_19_20`
   - The MCP process must see the library path. You will set `ORACLE_HOME` and `DYLD_LIBRARY_PATH` in Cursor's MCP config (section 3); if you start Cursor from a terminal that already has these set, you can reference them with `"ORACLE_HOME": "${env:ORACLE_HOME}"` etc.
   - To set in terminal for current session:
     ```bash
     export ORACLE_HOME=/opt/oracle/instantclient_19_20
     export DYLD_LIBRARY_PATH=$ORACLE_HOME:$DYLD_LIBRARY_PATH
     ```

---

## 2. Configure config.yaml

1. **Copy the example config**
   - In the folder where you extracted the zip, copy `config.yaml.example` and rename the copy to `config.yaml`

2. **Edit config.yaml**
   - Open `config.yaml` in any text editor.
   - Configure **oracle.connections** (required, at least one entry). Each key is a name you use in `execute_sql` as the `connection` argument; call `list_connections` to see names.

   ```yaml
   oracle:
     connections:
       database1: "username/password@//host:port/service_name"
       # Add more for multiple DBs (e.g. copy between servers):
       # database2: "user/pass@//host2:1521/SERVICE_NAME"
   ```

   - **One connection:** all SQL runs against that database; you don't need to pass `connection`.
   - **Multiple connections:** pass `"connection": "database1"` (or the name you configured) in `execute_sql` when calling the tool.

   Connection string format: `user/password@//host:port/service_name`  
   Examples: `scott/tiger@//localhost:1521/ORCL`, `myuser/mypass@//192.168.1.100:1521/PROD`.

   **Oracle Autonomous Database (ADB) with Wallet:** use the TNS alias from the wallet's `tnsnames.ora` (e.g. `mcpdemo_high`): `mcpdemo/YourPassword@mcpdemo_high`. You must set `TNS_ADMIN` to the wallet directory in MCP config (section 3).

   Other options (`security`, `logging`) are optional; you can leave them as-is.

3. **Save and location**
   - Keep `config.yaml` in the **same directory** as the executable, or set the environment variable `ORACLE_MCP_CONFIG` to the full path of your config file

---

## 3. Configure the MCP server in Cursor

1. **Open MCP settings**
   - In Cursor: **File** → **Preferences** → **Cursor Settings** → **MCP**
   - Or edit the config file directly:
     - **Windows:** `C:\Users\<YourUsername>\.cursor\mcp.json`
     - **macOS:** `~/.cursor\mcp.json`

2. **Add the Oracle MCP server**

   **Windows (basic)** — Instant Client already on system PATH:
   ```json
   {
     "mcpServers": {
       "oracle": {
         "command": "D:\\your_folder\\oracle-mcp-server-windows-amd64.exe",
         "args": []
       }
     }
   }
   ```
   Replace the `command` path with the **full path** to your executable (use `\\` for backslashes in JSON).

   **Windows + ADB Wallet** — set `TNS_ADMIN` to the folder where you unzipped the wallet (contains `tnsnames.ora`, `sqlnet.ora`, `cwallet.sso`, etc.), and ensure Instant Client is on `PATH` for the MCP process:
   ```json
   {
     "mcpServers": {
       "oracle": {
         "command": "D:\\your_folder\\oracle-mcp-server-windows-amd64.exe",
         "args": [],
         "env": {
           "TNS_ADMIN": "D:\\oracle\\wallet_mcpdemo",
           "PATH": "C:\\path\\to\\instantclient;%PATH%"
         }
       }
     }
   }
   ```
   Without `TNS_ADMIN`, you may see **ORA-12541** (no listener) or SSL errors.

   **macOS** — the MCP process must see `ORACLE_HOME` and `DYLD_LIBRARY_PATH` so it can load the Oracle libraries. Use the `env` block:
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
   Replace `/path/to/oracle-mcp-server-darwin-arm64` and `/opt/oracle/instantclient_19_20` with your actual paths. Use `oracle-mcp-server-darwin-amd64` on Intel Macs.

   **macOS + ADB Wallet** — also set `TNS_ADMIN` to your wallet directory (same folder as `tnsnames.ora` and wallet files):
   ```json
   "env": {
     "ORACLE_HOME": "/opt/oracle/instantclient_19_20",
     "DYLD_LIBRARY_PATH": "/opt/oracle/instantclient_19_20",
     "TNS_ADMIN": "/path/to/wallet_mcpdemo"
   }
   ```

3. **Restart Cursor**
   - Save `mcp.json`, then restart Cursor so the MCP server is loaded

4. **Verify**
   - If Oracle-related MCP tools (`execute_sql`, `execute_sql_file`, `list_connections`) appear in your chat, the setup is working
   - With multiple databases: call `list_connections` to see names, then use `execute_sql` with `"connection": "database1"` (or the name you configured) to run SQL on a specific database

---

## 4. Tools and behaviour

- **execute_sql** — Run SQL on the configured database(s). Params: `sql`, optional `connection`. When more than one connection is configured, pass `connection` with one of the names from `list_connections`. Dangerous or DDL statements open a **confirmation window** that shows the **database alias** and **operation type** (with extra spacing for clarity). You must confirm before execution.
- **execute_sql_file** — Read SQL from a file, apply the same review rules as `execute_sql`, then execute. Trailing SQL*Plus `/` is stripped. Params: `file_path`, optional `connection`.
- **list_connections** — List configured connection names and availability. Use these as the `connection` argument in `execute_sql` or `execute_sql_file`.

**Audit log** (`audit.log`, if enabled in config): each entry includes `CONNECTION=<alias>` so you can see which database was used (e.g. `CONNECTION=database1`, `CONNECTION=database2`).

---

## Troubleshooting

| Symptom | Likely cause | What to do |
|--------|----------------|------------|
| Error about missing oci.dll (Windows) | Oracle Instant Client not installed or not on PATH | Follow section 1; ensure Instant Client folder is on PATH. If using MCP `env`, set `PATH` to include it. |
| DPI-1047: Cannot locate a 64-bit Oracle Client library | Instant Client not found at runtime | Windows: add Instant Client to PATH (or to `env.PATH` in mcp.json). macOS: set `ORACLE_HOME` and `DYLD_LIBRARY_PATH` in mcp.json `env`. |
| ORA-12541: TNS:no listener / SSL or wallet errors | TNS name or Wallet not found | For ADB with Wallet, set `TNS_ADMIN` in mcp.json `env` to the folder containing `tnsnames.ora` and wallet files. |
| Error about missing config | config not found | Put config.yaml in the same folder as the binary, or set `ORACLE_MCP_CONFIG` to the config path. |
| Oracle tools not visible in Cursor | MCP not loaded or wrong path | Check the `command` path in mcp.json and restart Cursor. |

---

<a id="chinese"></a>

# Oracle MCP Server — 用户指南（中文）

**作者：** [Alvin Liu](https://alvinliu.com)

**项目：** [https://github.com/kjstart/cursor_oracle_mcp_server](https://github.com/kjstart/cursor_oracle_mcp_server)

本指南假设你已下载包含本平台可执行文件的发布包（如 `oracle-mcp-server-windows-amd64.exe`、`oracle-mcp-server-darwin-arm64` 或 `oracle-mcp-server-darwin-amd64`）以及 `config.yaml.example`，并已解压到本地。

---

## 1. 下载并配置 Oracle 客户端

1. **下载 Oracle Instant Client**
   - 打开：[Oracle Instant Client 下载页](https://www.oracle.com/database/technologies/instant-client/downloads.html)
   - 选择与系统匹配的包（Windows 64 位、macOS Intel/ARM），下载 **Basic** 包
   - 建议 19c 或更高版本
   - 所需文件：`oci.dll`、`oraociei19.dll`（Windows）或对应 `.dylib`（macOS）

2. **解压并设置环境**

   **Windows**
   - 将 zip 解压到某目录，如 `C:\oracle\instantclient_19_20`
   - 将该目录加入系统 **PATH**：
     - 右键 **此电脑** → **属性** → **高级系统设置** → **环境变量**
     - 在 **系统变量** 中选中 `Path`，点 **编辑** → **新建**，添加：`C:\oracle\instantclient_19_20`（请用你的实际路径）
   - **重启**终端或 Cursor 使新 PATH 生效

   **macOS**
   - 将 zip 解压到某目录，如 `/opt/oracle/instantclient_19_20`
   - MCP 进程需要能找到库路径。你将在 Cursor 的 MCP 配置（第 3 步）中设置 `ORACLE_HOME` 和 `DYLD_LIBRARY_PATH`；若从已设置这些变量的终端启动 Cursor，可在配置中用 `"ORACLE_HOME": "${env:ORACLE_HOME}"` 等引用。
   - 在当前终端会话中可执行：
     ```bash
     export ORACLE_HOME=/opt/oracle/instantclient_19_20
     export DYLD_LIBRARY_PATH=$ORACLE_HOME:$DYLD_LIBRARY_PATH
     ```

---

## 2. 配置 config.yaml

1. **复制示例配置**
   - 在解压目录中，将 `config.yaml.example` 复制一份并重命名为 `config.yaml`

2. **编辑 config.yaml**
   - 用任意文本编辑器打开 `config.yaml`。
   - 配置 **oracle.connections**（必填，至少一项）。每个键是你在 `execute_sql` 中用作 `connection` 参数的名称；可调用 `list_connections` 查看名称。

   ```yaml
   oracle:
     connections:
       database1: "用户名/密码@//主机:端口/服务名"
       # 多库时可继续添加，例如：
       # database2: "user/pass@//host2:1521/SERVICE_NAME"
   ```

   - **单连接：** 所有 SQL 都发往该数据库，无需传 `connection`。
   - **多连接：** 在调用工具时传入 `"connection": "database1"`（或你配置的名称）。

   连接串格式：`user/password@//host:port/service_name`  
   示例：`scott/tiger@//localhost:1521/ORCL`，`myuser/mypass@//192.168.1.100:1521/PROD`。

   **Oracle 自治数据库 (ADB) + Wallet：** 使用钱包中 `tnsnames.ora` 的 TNS 别名（如 `mcpdemo_high`）：`mcpdemo/YourPassword@mcpdemo_high`。必须在 MCP 配置（第 3 步）中设置 `TNS_ADMIN` 为 Wallet 所在目录。

   其他选项（`security`、`logging`）可选，可保持默认。

3. **保存与位置**
   - 将 `config.yaml` 放在与可执行文件 **同一目录**，或设置环境变量 `ORACLE_MCP_CONFIG` 指向配置文件的完整路径

---

## 3. 在 Cursor 中配置 MCP 服务

1. **打开 MCP 设置**
   - Cursor：**File** → **Preferences** → **Cursor Settings** → **MCP**
   - 或直接编辑配置文件：
     - **Windows：** `C:\Users\<你的用户名>\.cursor\mcp.json`
     - **macOS：** `~/.cursor/mcp.json`

2. **添加 Oracle MCP 服务**

   **Windows（基础）** — Instant Client 已在系统 PATH 中：
   ```json
   {
     "mcpServers": {
       "oracle": {
         "command": "D:\\你的目录\\oracle-mcp-server-windows-amd64.exe",
         "args": []
       }
     }
   }
   ```
   将 `command` 的路径替换为你的可执行文件 **完整路径**（JSON 中反斜杠用 `\\`）。

   **Windows + ADB Wallet** — 将 `TNS_ADMIN` 设为 Wallet 解压目录（内含 `tnsnames.ora`、`sqlnet.ora`、`cwallet.sso` 等），并确保 MCP 进程的 `PATH` 中包含 Instant Client：
   ```json
   {
     "mcpServers": {
       "oracle": {
         "command": "D:\\你的目录\\oracle-mcp-server-windows-amd64.exe",
         "args": [],
         "env": {
           "TNS_ADMIN": "D:\\oracle\\wallet_mcpdemo",
           "PATH": "C:\\path\\to\\instantclient;%PATH%"
         }
       }
     }
   }
   ```
   未设置 `TNS_ADMIN` 可能出现 **ORA-12541**（无监听）或 SSL 错误。

   **macOS** — MCP 进程必须能读取 `ORACLE_HOME` 和 `DYLD_LIBRARY_PATH` 以加载 Oracle 库。使用 `env` 块：
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
   将 `/path/to/oracle-mcp-server-darwin-arm64` 和 `/opt/oracle/instantclient_19_20` 换成你的实际路径。Intel Mac 使用 `oracle-mcp-server-darwin-amd64`。

   **macOS + ADB Wallet** — 在 `env` 中再设置 `TNS_ADMIN` 为 Wallet 目录（与 `tnsnames.ora` 及钱包文件同目录）：
   ```json
   "env": {
     "ORACLE_HOME": "/opt/oracle/instantclient_19_20",
     "DYLD_LIBRARY_PATH": "/opt/oracle/instantclient_19_20",
     "TNS_ADMIN": "/path/to/wallet_mcpdemo"
   }
   ```

3. **重启 Cursor**
   - 保存 `mcp.json` 后重启 Cursor，以加载 MCP 服务

4. **验证**
   - 若对话中出现 Oracle 相关 MCP 工具（`execute_sql`、`execute_sql_file`、`list_connections`），说明配置成功
   - 多数据库时：先调用 `list_connections` 查看名称，再在 `execute_sql` 中传入 `"connection": "database1"`（或你配置的名称）对指定库执行 SQL

---

## 4. 工具与行为

- **execute_sql** — 在已配置的数据库上执行 SQL。参数：`sql`，可选 `connection`。配置了多个连接时，传入 `list_connections` 返回的名称之一作为 `connection`。危险或 DDL 语句会弹出 **确认窗口**，显示 **数据库别名** 和 **操作类型**，需确认后才会执行。
- **execute_sql_file** — 从文件读取 SQL，应用与 `execute_sql` 相同的审查规则后执行。末尾 SQL*Plus 的 `/` 会被去除。参数：`file_path`，可选 `connection`。
- **list_connections** — 列出已配置连接名称及可用性。可将返回的名称作为 `execute_sql` 或 `execute_sql_file` 的 `connection` 参数。

**审计日志**（若在配置中启用 `audit.log`）：每条记录包含 `CONNECTION=<别名>`，便于查看使用的数据库（如 `CONNECTION=database1`、`CONNECTION=database2`）。

---

## 故障排除

| 现象 | 可能原因 | 处理 |
|--------|----------------|------------|
| 报错缺少 oci.dll（Windows） | 未安装 Oracle Instant Client 或未加入 PATH | 按第 1 步操作；确保 Instant Client 目录在 PATH 中。若在 MCP 的 `env` 中配置，则在该处设置 `PATH` 包含该目录。 |
| DPI-1047: Cannot locate a 64-bit Oracle Client library | 运行时找不到 Instant Client | Windows：将 Instant Client 加入 PATH（或在 mcp.json 的 `env.PATH` 中设置）。macOS：在 mcp.json 的 `env` 中设置 `ORACLE_HOME` 和 `DYLD_LIBRARY_PATH`。 |
| ORA-12541: TNS:no listener / SSL 或钱包相关错误 | 找不到 TNS 名或 Wallet | 使用 ADB + Wallet 时，在 mcp.json 的 `env` 中设置 `TNS_ADMIN` 为包含 `tnsnames.ora` 和钱包文件的目录。 |
| 报错找不到 config | 未找到配置文件 | 将 config.yaml 放在与可执行文件同一目录，或设置 `ORACLE_MCP_CONFIG` 指向配置文件路径。 |
| Cursor 中看不到 Oracle 工具 | MCP 未加载或路径错误 | 检查 mcp.json 中 `command` 路径是否正确，并重启 Cursor。 |
