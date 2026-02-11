# Oracle MCP Server — User Guide

This guide assumes you have downloaded the zip containing `oracle-mcp.exe` and `config.yaml.example` and extracted it locally.

---

## 1. Download and configure Oracle Client

1. **Download Oracle Instant Client**
   - Go to: [Oracle Instant Client downloads](https://www.oracle.com/database/technologies/instant-client/downloads.html)
   - Choose the package that matches your system (e.g. Windows 64-bit) and download the **Basic** package
   - Version 19c or later is recommended

2. **Extract and add to PATH**
   - Extract the downloaded zip to a folder, e.g. `C:\oracle\instantclient_19_20`
   - Add that folder to your system **PATH**:
     - Right-click **This PC** → **Properties** → **Advanced system settings** → **Environment variables**
     - Under **System variables**, select `Path`, click **Edit**, then **New** and add: `C:\oracle\instantclient_19_20` (use your actual path)
   - **Restart** your terminal or Cursor so the new PATH takes effect

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

   Other options (`security`, `logging`) are optional; you can leave them as-is.

3. **Save and location**
   - Keep `config.yaml` in the **same directory** as `oracle-mcp.exe`, or set the environment variable `ORACLE_MCP_CONFIG` to the full path of your config file

---

## 3. Configure the MCP server in Cursor

1. **Open MCP settings**
   - In Cursor: **File** → **Preferences** → **Cursor Settings** → **MCP**
   - Or edit the config file directly: `C:\Users\<YourUsername>\.cursor\mcp.json`

2. **Add the Oracle MCP server**
   - Add an entry under `mcpServers`, for example:
   ```json
   {
     "mcpServers": {
       "oracle": {
         "command": "D:\\your_folder\\oracle-mcp.exe",
         "args": []
       }
     }
   }
   ```
   - Replace `"D:\\your_folder\\oracle-mcp.exe"` with the **full path** to your `oracle-mcp.exe` (use `\\` for backslashes in JSON on Windows)

3. **Restart Cursor**
   - Save `mcp.json`, then restart Cursor so the MCP server is loaded

4. **Verify**
   - If Oracle-related MCP tools (`execute_sql`, `list_connections`) appear in your chat, the setup is working
   - With multiple databases: call `list_connections` to see names, then use `execute_sql` with `"connection": "database1"` or `"connection": "database2"` to run SQL on a specific database

---

## 4. Tools and behaviour

- **execute_sql** — Run SQL on the configured database(s). When more than one connection is configured, pass `connection` with one of the names from `list_connections`. Dangerous or DDL statements open a **confirmation window** that shows the **database alias** and **operation type** (with extra spacing for clarity). You must confirm before execution.
- **list_connections** — List configured connection names. Use these as the `connection` argument in `execute_sql`.

**Audit log** (`audit.log`, if enabled in config): each entry includes `CONNECTION=<alias>` so you can see which database was used (e.g. `CONNECTION=database1`, `CONNECTION=database2`).

---

## Troubleshooting

| Symptom | Likely cause | What to do |
|--------|----------------|------------|
| Error about missing oci.dll | Oracle Instant Client not installed or not on PATH | Follow section 1 and check extract path and PATH |
| Error about missing config | config.yaml not next to the exe | Put config.yaml in the same folder as the exe, or set `ORACLE_MCP_CONFIG` to the config path |
| Oracle tools not visible in Cursor | MCP not loaded or wrong path | Check the `command` path in mcp.json and restart Cursor |
