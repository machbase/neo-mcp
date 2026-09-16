# neo-mcp

`neo-mcp` exposes machbase-neo database, TQL, JSH, Markdown, and server-side file operations to an MCP client such as VS Code Copilot.

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
        "-max-response-bytes", "2097152"
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

`-max-response-bytes` limits each HTTP response read by neo-mcp. The default is 2 MiB (`2097152` bytes). Set it explicitly in `mcp.json` when a workload needs a different limit, for example `10485760` for 10 MiB. Values less than or equal to zero are rejected.

## Tools

- `db_query`: execute SQL with HTTP query options such as `format`, `timeformat`, `tz`, `db`, and bind parameters.
- `tql_run` and `tql_run_file`: execute inline TQL or a server-side `.tql` file.
- `jsh_exec` and `jsh_run_command`: execute JSH through the SSH service.
- `render_markdown`: render Markdown and execute executable SQL, JSH, and HTTP fences according to the Markdown contract.
- `fs_list`, `fs_read`, and `fs_write`: inspect and modify the server-side SSFS through the API-token `/db/files/*path` endpoint.

Server-side file paths are not local workspace paths. Use `fs_list` to discover files before using `fs_read`, `fs_write`, or `tql_run_file`.

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
