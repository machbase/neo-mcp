# JSH `http` Server Module

Reference implementation: https://github.com/machbase/neo-demo/tree/main/demos/server
(companion DB demo: https://github.com/machbase/neo-demo/tree/main/demos/machcli)

`http` provides a Node-like server API for JSH web applications. Load it with
`require('http')`.

## Minimal server

```js
'use strict';
const http = require('http');
const process = require('process');

const server = new http.Server({
    network: 'tcp',
    address: '0.0.0.0:7575',
    env: process.env,
});

server.get('/', (ctx) => {
    ctx.json(http.status.OK, { message: 'hello' });
});

server.serve((result) => {
    console.println('server started', result.network, result.address);
});
```

- `new http.Server({ network, address, env })` creates the listener; `env`
  defaults to the current `process.env`.
- Route registration: `server.get(path, ctx => {...})`, `.post`, `.put`,
  `.delete`. Handlers receive one `ctx` argument.
- `ctx.query(name)` reads a query-string parameter; `ctx.param(name)` reads a
  path parameter; `ctx.request.body` holds the parsed request body.
- Response helpers: `ctx.json(status, data[, opt])`, `ctx.html(status,
  templateName, data)`, and lower-level `ctx.render(...)`.
- `server.static(routePath, rootDir)` mounts a static file directory;
  `server.staticFile(routePath, file)` mounts a single file.
- `server.loadHTMLFiles(...files)` / `server.loadHTMLGlob(pattern)` register
  HTML templates for `ctx.html(...)`. These are Go `html/template` files
  (backed by `gin`'s `LoadHTMLFiles`/`LoadHTMLGlob`), using `{{.field}}` and
  `{{range .items}}...{{end}}` syntax — **not** real Mustache `{{#section}}`
  syntax, even though the neo-demo example names its files `*.mustache.html`.
  There is no `mustache` JSH module in this neo-server build; do not generate
  a script that calls `require('mustache')`.
- `server.ws(path, handler)` attaches a WebSocket route (backed by the `ws`
  module).

## Execution model warning

`server.serve(callback)` starts listening and keeps the JSH process alive with
an internal timer for as long as `listening` is `true`; it does not return
control back to a synchronous script. `jsh_exec` and `jsh_run_file` run a
script through a single SSH exec and wait for that process to exit, so
invoking an entry script that calls `server.serve()` (and never
`server.close()`) through either tool will hang the MCP tool call
indefinitely.

Do not call `jsh_exec`/`jsh_run_file` on a script whose top level starts a
server with `serve()`. Instead:

1. Use `fs_write` to create the application's files under `/project` (see
   layout below), so the user or the service controller can start it outside
   the one-shot MCP execution tools.
2. If a quick smoke test is needed, test route handlers as plain functions or
   call `server.close()` right after a short-lived `serve()`/`close()` pair in
   a bounded script, rather than leaving the server running.
3. Tell the user how to start the written application themselves (for
   example, running it as a background process, or registering it with
   `require('service')` as shown in `install.js` below) rather than claiming
   the MCP tool started a long-lived server.

## Application file layout

Mirror the demo's file layout when writing a non-trivial JSH web application
under `/project/<app-name>/`. `fs_write` creates any missing parent
directories automatically (`mkdir -p` semantics), so nested paths like
`/project/my-app/handlers/tags.js` can be written directly without a
separate mkdir step:

- `index.js` — entry point: parses `--port`, builds `http.Server`, registers
  `server.static(...)` and one `server.get/post/...` call per route (usually
  delegating to a handler module), then calls `server.serve(...)`.
- `support.js` — shared helpers such as `parsePort(argv)`,
  `resolveScriptDir()` (from `process.argv[1]`), and `loadTemplate(fileName)`.
- `handlers/*.js` — one module per route, each exporting a function that takes
  `ctx` and calls a `ctx.*` response helper.
- `*.mustache.html` — one Go `html/template` file per page (see the
  template-syntax note above despite the `.mustache.html` extension), loaded
  via `server.loadHTMLFiles(...)` and rendered with `ctx.html(status,
  templateName, data)`.
- `static/` — static assets mounted with `server.static('/static', dir)`.
- `install.js` — optional: registers `index.js` as a machbase-neo-managed
  service via `require('service').Client().install({ name, executable, args
  })`, so the service controller (not a one-shot MCP call) starts/stops the
  server. Only write and explain this file; do not call `service.Client()`
  install/start actions from an MCP tool without explicit user approval, since
  service changes are a shared-system side effect.

```js
// support.js
const path = require('path');
function resolveScriptDir() {
    if (!process.argv[1]) return process.cwd();
    return path.dirname(process.argv[1]);
}
function parsePort(argv) {
    for (let i = 0; i < argv.length; i += 1) {
        if (argv[i] === '--port') return Number(argv[i + 1]);
    }
    return 7575;
}
module.exports = { resolveScriptDir, parsePort };
```

## Combining with `machcli`

A handler module can open its own `machcli` connection per request (or reuse a
module-level `Client`/pool) to serve database-backed pages. Follow
`neo://manual/jsh/machcli` for connection, query, and cleanup patterns; always
close `rows`/`conn` in a `finally` block inside the handler so a failed
request does not leak connections across many requests.

## Instant app workflow (target and current gap)

The target end-to-end flow for a data-analysis/processing request is:

1. User asks for data analysis or processing.
2. The LLM writes a JSH web app (this document's layout) with `fs_write`.
3. The LLM starts the app's server inside the machbase-neo JSH environment.
4. The LLM opens the running app in the internal/integrated browser (falling
   back to a plain link, per `neo://manual/tql`'s "Opening a result URL"
   rule), so the user sees an interactive, GUI-driven result rather than a
   static text/chart dump.
5. The user explores the result (follows links, submits query forms) directly
   in the running app.

Step 3 is not yet achievable through the current MCP toolset. `jsh_exec`,
`jsh_run_file`, and `jsh_run_command` all run a script or command line
through a single synchronous SSH exec and wait for the process to exit; there
is no MCP tool that starts a detached/background jsh process. Inside JSH
itself, `process.exec()` is also synchronous (it waits for the child to
exit), so a script cannot daemonize its own server process either. Calling
`server.serve()` from any of these tools blocks that tool call indefinitely
(see "Execution model warning" above).

Until a background-start capability exists (for example, a future MCP tool
wrapping the `service` module's install/start calls, see
`neo://manual/jsh/service`), stop at step 2 and hand off steps 3-5 like this:

1. Write the app with `fs_write` as usual.
2. Tell the user the exact command to start it themselves (for example,
   `jsh /work/<app>.index.js --port <port>` in their own neo-shell/terminal),
   and wait for their confirmation that it is listening.
3. Once confirmed, open the app's URL in the internal browser (or present the
   link as a fallback) exactly as if it were a chart/TQL result link.

Do not claim the LLM started the server unless a tool call actually did so
without hanging. Revisit this section if a background-execution or
service-install MCP tool becomes available, and update the workflow to run
step 3 automatically instead of asking the user to start the app.
