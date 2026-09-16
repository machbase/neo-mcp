# Machbase JSH Guidance

Use the `jsh_exec` and `jsh_run_command` tools to execute JSH through the existing machbase-neo SSH interface. The two tools connect with different SSH identities and give different runtime environments:

- `jsh_run_command` connects with the reserved `neo-mcp` SSH username and enters full neo-shell. This shell has already logged into the database (`NEOSHELL_*` env), mounts `/usr/bin` and `/usr/lib`, and exposes DB client commands (`sql`, `show`, `import`, `export`). Use it to run an existing neo-shell/JSH command line, including the neo-shell `jsh` alias for interactive-style usage.
- `jsh_exec` connects with the reserved `neo-mcp:jsh` SSH username, which selects the raw `jsh` shellId directly (bypassing neo-shell). The SSH exec command is `-C "<script>"`, handled by the jsh engine's own `-C` flag: the script runs once and the process exits. This is required because neo-shell's `jsh` alias (`/sbin/shell.js`) ignores `-C` and always starts an interactive REPL, which hangs forever on a non-interactive SSH exec.

**Important trade-off for `jsh_exec`:** the raw `jsh` shellId has no DB session, no `NEOSHELL_*` login, and no `/usr/bin` commands — only `/sbin` (system commands) and `/lib` (native `@jsh/*` modules) are mounted. A script that needs the database must open its own connection explicitly (for example `require('@jsh/machcli')` with credentials), or use `jsh_run_command` instead if it depends on an already-authenticated session or `/usr/bin` commands.

## Runtime constraints

- JSH runs JavaScript in the Goja-based runtime, not in Node.js or a browser.
- Do not use `await` or `import`.
- Use `require()` for modules.
- `Buffer` and `URL` are available from the runtime.
- Use `console.log`, `console.println`, and related console methods for diagnostics.

## Runtime globals

The JSH language service documents these common globals:

- `require`
- `console`
- `process`
- `Buffer`
- `URL`

## Modules and APIs

The canonical module and export metadata comes from the neo-server JSH language service:

- `neo-server/mods/lsp/jsh/service.go` builds runtime and module metadata.
- `neo-server/mods/lsp/jsh/jsdoc.go` extracts exported functions/classes and JSDoc signatures.
- `neo-server/jsh/lib/` contains native and JavaScript module implementations.
- `neo-server/jsh/usr/bin/` contains database-oriented commands such as `sql`, `show`, `import`, and `export`.

Before writing a non-trivial script, inspect the available module/export metadata and follow the documented signature instead of guessing a Node.js API.

## Execution workflow

1. `jsh_exec` (inline script): connects as `neo-mcp:jsh` and runs `-C "<script>"` through the raw jsh engine — one-shot execution, no DB session, no `/usr/bin`.
2. `jsh_run_command` (existing command): connects as `neo-mcp` and runs the given command line inside full neo-shell — DB session and `/usr/bin` commands (`sql`, `show`, `import`, `export`) are available.
3. Keep stdout useful and concise so the result can be interpreted by the agent.
4. Treat file, network, database, and shell side effects as real operations controlled by the token owner's permissions.
