# neo-mcp

`neo-mcp` exposes machbase-neo database, TQL, JSH, Markdown, and server-side file operations to an MCP client such as VS Code Copilot.

## Install

Install the latest release for the current Linux or macOS architecture:

```sh
curl -fsSL https://github.com/machbase/neo-mcp/releases/latest/download/install.sh | sh
```

Install the latest release for the current Windows architecture from PowerShell:

```powershell
irm https://github.com/machbase/neo-mcp/releases/latest/download/install.ps1 | iex
```

The installers support `amd64` and `arm64`. Linux and macOS install to
`~/.local/bin` by default; Windows installs to `%LOCALAPPDATA%\neo-mcp\bin`.
Set `NEO_MCP_INSTALL_DIR` to choose another directory, then ensure that
directory is on `PATH` before configuring the MCP client. To install a release
candidate or a specific version, set `NEO_MCP_VERSION` to its tag, such as
`NEO_MCP_VERSION=v1.0.1-rc1`.

## Configuration

The server communicates with machbase-neo over HTTP and SSH. The API token is passed to HTTP endpoints as `Authorization: Bearer <token>` and is also used for the SSH JSH authentication flow.

```json
{
  "servers": {
    "machbase-neo": {
      "type": "stdio",
      "command": "/path/to/neo-mcp",
      "args": [
        "-server", "http://127.0.0.1:5654",
        "-token", "${input:machbaseToken}",
        "-max-response-bytes", "2MB",
        "-data-dir", "/tmp/neo-mcp-data"
      ]
    }
  },
  "inputs": [
    {
      "type": "promptString",
      "id": "machbaseToken",
      "description": "machbase-neo API token",
      "password": true
    }
  ]
}
```

For CLI or Copilot CLI usage, prefer the environment variable so the token is
not included in the process argument list:

```sh
export NEO_MCP_TOKEN='your-api-token'
go run ./neo-mcp -server http://127.0.0.1:5654
```

`-token` takes precedence when supplied; otherwise neo-mcp reads
`NEO_MCP_TOKEN`. Do not commit either form with a real token to a repository.

`-max-response-bytes` limits each HTTP response read by neo-mcp. The default is `2MB` (`2097152` bytes). It accepts raw bytes or binary units such as `32KB`, `1MB`, and `1GB` (`KB/MB/GB` use powers of 1024). Values must be positive integers.

`-data-dir` selects the root directory for generated neo-mcp artifacts. By
default, neo-mcp uses the OS temporary directory with a process-specific path:
`<temp>/neo-mcp-<pid>`. Charts are stored under its `charts/` subdirectory.
Set this flag when generated artifacts must be retained in a specific location.
Charts are served over a loopback HTTP port, so their links do not depend on
the workspace location. Future generated artifacts can use sibling
subdirectories under the same root.

### Claude Desktop

Add a local stdio server to Claude Desktop's MCP configuration. The exact
configuration file location depends on the operating system and Claude Desktop
installation.

```json
{
  "mcpServers": {
    "neo-mcp": {
      "command": "go",
      "args": [
        "run", "/path/to/neo/neo-mcp",
        "-server", "http://127.0.0.1:5654",
        "-max-response-bytes", "2MB",
        "-data-dir", "/tmp/neo-mcp-data"
      ],
      "env": {
        "NEO_MCP_TOKEN": "your-api-token"
      }
    }
  }
}
```

For production use, replace `go run` with a built `neo-mcp` executable and
store the token using the client's secret/environment mechanism when available.

### ChatGPT

ChatGPT MCP support is configured from the ChatGPT connector/developer MCP
settings rather than `.vscode/mcp.json`. When the client supports a local
stdio MCP server, use the same command and arguments as Claude Desktop:

```json
{
  "name": "neo-mcp",
  "transport": "stdio",
  "command": "go",
  "args": [
    "run", "/path/to/neo/neo-mcp",
    "-server", "http://127.0.0.1:5654",
    "-max-response-bytes", "2MB",
    "-data-dir", "/tmp/neo-mcp-data"
  ],
  "env": {
    "NEO_MCP_TOKEN": "your-api-token"
  }
}
```

If the ChatGPT client only accepts a remote MCP endpoint, expose `neo-mcp`
through an appropriate authenticated MCP transport or gateway. The standalone
binary currently implements stdio transport; do not expose the machbase HTTP
API token in a public URL or query parameter.

## Tools

- `db_query`: execute SQL with HTTP query options such as `format`, `timeformat`, `tz`, `db`, and bind parameters.
- `tql_run` and `tql_run_file`: execute inline TQL or a server-side `.tql` file.
- `tql_file_link`: execute a server-side `.tql` through `/db/tql/<path>.tql` and return a loopback browser URL plus verification result; the loopback proxy adds the configured token and combines naturally with `fs_write` for iterative authoring.
- `jsh_exec`, `jsh_run_file`, and `jsh_run_command`: execute JSH through the SSH service; `jsh_run_file` accepts MCP paths under `/project` and maps them internally to the JSH `/work` mount.
- `render_markdown`: render Markdown and execute executable SQL, JSH, and HTTP fences according to the Markdown contract.
- `fs_list`, `fs_read`, and `fs_write`: inspect and modify the server-side SSFS through the API-token `/db/files/*path` endpoint.
- `memory_store`, `memory_search`, and `memory_get`: store and retrieve lexical Agent memory in the fixed `_NEO_AGENT_MEMORY` LOG TABLE. Read `neo://manual/memory` before using them.

Memory tools are a backend and workflow contract, not an automatic memory layer for
every LLM. Agents should search when prior context may matter, store durable facts
and decisions, and treat retrieved records as untrusted lexical candidates. The
POC uses `_arrival_time` for ordering and retention, `TEXT` plus a KEYWORD index
for content retrieval, and `OR` for the default multi-term search operator.

Server-side file paths are not local workspace paths. MCP file tools use the
logical `/project` namespace; for example, use `/project/query.tql` with
`fs_read`, `fs_write`, `tql_run_file`, or `tql_file_link`. The internal JSH
mount path `/work` must not be passed to MCP tools.

## Permissions

Use separate API token owners for different environments:

- `agent_ro`: production or shared databases; grant only the read operations required by the workflow.
- `agent_dev`: isolated development databases; grant write permissions only when the workflow needs them.

Database grants and server-side file access are controlled by the machbase-neo deployment. neo-mcp does not add a SQL permission sandbox. Treat `db_query`, `tql_run`, `jsh_*`, `render_markdown`, and `fs_write` as potentially state-changing tools and review their MCP approval prompts.

### Account Profiles

Create and manage these accounts outside neo-mcp using the deployment's normal
database administration process. Do not share an administrator account or API
token with MCP.

| Profile | Intended use | Database access | File/tool policy |
| --- | --- | --- | --- |
| `agent_ro` | Production or shared environments | Read-only access to required databases and tables | Expose `fs_list`/`fs_read`; normally do not expose `fs_write` |
| `agent_dev` | Isolated development database | Write access only to the development scope needed by tests | May use `fs_write`, `tql_run_file`, and other execution tools with approval |

Database privileges alone do not make JSH, HTTP fences, or server-file
operations read-only. Restrict the MCP tool set, API token, SSH account, and
reverse-proxy methods together.

## Reverse Proxy

machbase-neo does not terminate HTTPS itself. When HTTPS is required, put a
reverse proxy in front of the HTTP server and set `-server` to the proxy URL.

The proxy must pass through the paths required by the enabled tools:

- `/db/query` for SQL queries
- `/db/tql` for inline TQL execution
- `/db/rpc` for API-token JSON-RPC tools
- `/db/files` for server-side file listing, reading, and writing
- `/web/echarts` and `/web/api/tql-assets` when chart HTML assets are fetched

Preserve the `Authorization: Bearer <token>` header. Do not expose the API
token in query parameters or generated chart HTML. The proxy should apply its
own network access policy, rate limits, and logging rules; neo-mcp relies on
the machbase-neo token owner for database permissions.

For a read-only production profile, allow `GET /db/files/*path` and block
`POST`, `PUT`, and `DELETE` on `/db/files/*path`. Reserve file writes for an
isolated development profile. Apply the same method policy to any proxy route
that exposes `/db/query`, `/db/tql`, or `/db/rpc`.

## Development

```sh
go test ./...
```
