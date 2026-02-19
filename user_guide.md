# Oracle MCP Server — User Guide

**Author:** [Alvin Liu](https://alvinliu.com)

**GitHub:** [cursor_oracle_mcp_server](https://github.com/kjstart/cursor_oracle_mcp_server)

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
   - The MCP process must see the library path. You will set `ORACLE_HOME` and `DYLD_LIBRARY_PATH` in Cursor’s MCP config (section 3); if you start Cursor from a terminal that already has these set, you can reference them with `"ORACLE_HOME": "${env:ORACLE_HOME}"` etc.
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

   - **One connection:** all SQL runs against that database; you don’t need to pass `connection`.
   - **Multiple connections:** pass `"connection": "database1"` (or the name you configured) in `execute_sql` when calling the tool.

   Connection string format: `user/password@//host:port/service_name`  
   Examples: `scott/tiger@//localhost:1521/ORCL`, `myuser/mypass@//192.168.1.100:1521/PROD`.

   **Oracle Autonomous Database (ADB) with Wallet:** use the TNS alias from the wallet’s `tnsnames.ora` (e.g. `mcpdemo_high`): `mcpdemo/YourPassword@mcpdemo_high`. You must set `TNS_ADMIN` to the wallet directory in MCP config (section 3).

   Other options (`security`, `logging`) are optional; you can leave them as-is.

3. **Save and location**
   - Keep `config.yaml` in the **same directory** as the executable, or set the environment variable `ORACLE_MCP_CONFIG` to the full path of your config file

---

## 3. Configure the MCP server in Cursor

1. **Open MCP settings**
   - In Cursor: **File** → **Preferences** → **Cursor Settings** → **MCP**
   - Or edit the config file directly:
     - **Windows:** `C:\Users\<YourUsername>\.cursor\mcp.json`
     - **macOS:** `~/.cursor/mcp.json`

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
