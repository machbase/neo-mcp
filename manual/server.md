# Machbase Server Management Guidance

These tools call machbase-neo's JSON-RPC methods through the API-token authenticated
`/db/rpc` endpoint (as opposed to `db_query`/`tql_run`, which use `/db/query` and
`/db/tql`). Only a fixed allowlist of methods is reachable this way
(`dbRpcAllowedMethods` in neo-server/mods/server/http.go); every method in that
allowlist is scoped to the API token's owner, so these tools only ever see or
manage resources owned by (or explicitly shared with) that user — never another
user's timers, subscribers, or tokens.

## Tools

- `render_markdown` — render Markdown text to HTML and execute fenced blocks.
  `sql` and `jsh` blocks require `{execute=true}`; `http` blocks execute by the
  established Markdown contract. Treat this as a side-effecting operation.
- `server_info` — read-only version/runtime info (pid, uptime, memory). No arguments.
- `timer_list` / `timer_get` / `timer_add` / `timer_update` / `timer_delete` /
  `timer_start` / `timer_stop` — manage cron-style timer schedules owned by the caller.
- `subscriber_list` / `subscriber_get` / `subscriber_add` / `subscriber_update` /
  `subscriber_delete` / `subscriber_start` / `subscriber_stop` — manage MQTT bridge
  subscribers owned by the caller. NATS subscriber options exist server-side but are
  not exposed by these tools; use `jsh_run_command` for NATS subscribers.
- `token_list` / `token_generate` / `token_delete` — manage the caller's own API
  tokens. `token_generate` returns the plaintext token value exactly once; it cannot
  be retrieved again afterward.

## Notes

- `timer_add`/`subscriber_add` create schedules that execute a shell command string
  on trigger; treat the `command` argument as a real, side-effecting operation under
  the token owner's OS/DB permissions, not a sandboxed preview.
- `token_generate`/`token_delete` change the caller's own authentication material.
  Deleting a token invalidates it immediately for every client using it, including
  the MCP session's own token if it happens to be the one deleted.
- These tools are unrelated to `db_query`/`tql_run`/`jsh_exec`/`jsh_run_command`,
  which use different HTTP/SSH paths documented in neo://manual/sql,
  neo://manual/tql, and neo://manual/jsh.
