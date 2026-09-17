# Machbase TQL Guidance

Use the `tql_run` and `tql_run_file` MCP tools to execute TQL through the existing machbase-neo `/db/tql` HTTP API.

## Agent execution contract

- After writing a TQL script, call `tql_run` or `tql_run_file` to execute it.
- Do not execute TQL by opening a terminal, running `curl`, or calling `/db/tql` directly. The MCP Tool is the supported execution interface because it preserves the agent tool trace and returns the result in the MCP conversation.
- For a request such as "write and execute TQL", the required sequence is: `manual_read(neo://manual/tql)` -> write the script -> `tql_run(script)` -> explain the result.
- Do not claim that a script ran until the `tql_run` Tool has returned a result.

## Flow model

A TQL script normally has a source, zero or more map/transform stages, and a sink:

```tql
FAKE(linspace(1, 3, 3))
MAPVALUE(0, value(0) * 10)
CSV()
```

- Sources produce records, for example `SQL()`, `SQL_SELECT()`, `FAKE()`, `CSV()`, and `JSON()`.
- Map stages transform records, for example `MAPVALUE()` and expression helpers.
- Sinks encode or write records, for example `CSV()`, `JSON()`, `CHART()`, `APPEND()`, and `INSERT()`.

## Authoring rules

- Prefer a small source and a bounded result while iterating.
- Use `CSV()` or `JSON()` for machine-readable inspection.
- Use `CHART()` for ECharts-based chart output; do not invent a separate chart protocol.
- Use backtick strings for multi-line SQL passed to `SQL()`.
- Use `param()` for HTTP query parameters and `value()`/`key()` for record fields.
- Validate source/map/sink structure before retrying a runtime failure.
- When a request asks for "a demo chart" from an unspecified tag/time window
  rather than a specific one, do not just pick the first tag and the first
  time range. Query aggregate stats (`MIN`, `MAX`, `AVG`, `STDDEV`) per
  candidate `NAME`/time window first via `db_query`, and pick a tag and
  window with a meaningfully non-zero `STDDEV_VALUE`. A flat or near-constant
  series (low `STDDEV`) makes a poor visualization demo even though the query
  itself succeeds.

## Minimal execution example

For a JSON result from the `EXAMPLE` table, write and execute this through `tql_run`:

```tql
SQL(`SELECT TIME, VALUE FROM EXAMPLE LIMIT 20`)
JSON()
```

## CHART and ECharts embedding

Canonical embedding guide: https://docs.machbase.com/neo/tql/chart/embed_in_html.md

Use `CHART()` when the requested result is a visualization. A typical pattern is:

```tql
SQL(`SELECT TIME, VALUE FROM EXAMPLE LIMIT 100`)
CHART(
	chartOption({
		xAxis: { type: "time" },
		yAxis: {},
		series: [{ type: "line", data: column(1) }]
	})
)
```

The `tql_run` Tool requests the Machbase chart JSON envelope. It contains the chart id, dimensions, ECharts assets, and generated chart code assets. A VS Code extension/webview renderer can load `jsAssets` first and then `jsCodeAssets`; the generated code must be loaded after ECharts finishes loading. Do not treat the chart envelope as ordinary tabular JSON, and do not replace `CHART()` with a hand-written chart protocol.

### Multi-series charts

`column(idx)` inside `chartOption({...})` takes exactly one argument and
returns the full array of that record column's values (`_columns[idx]`); it
does **not** accept a second argument to zip an x/y pair. `column(0, 1)`
silently ignores the `1` and returns the same array as `column(0)`. Using it
for two series' `data` therefore plots two identical, overlapping lines with
no visible variation and an unhelpful legend.

To plot one series per selected column against a shared `TIME` column (column
0), build `[x, y]` pairs explicitly with `.map()`, and name each series so the
`legend` can reference it. When both `title` and `legend` are present, also
set explicit positions for them: both default to `top: 'auto'`/`left: 'auto'`,
which places both in the top-left corner and makes them overlap. Push the
legend down with `legend.top` and grow `grid.top` by a matching amount so the
plot area does not get covered:

```tql
SQL(`SELECT a.TIME, a.VALUE AS temperature, b.VALUE AS dew_point
FROM EXAMPLE a, EXAMPLE b
WHERE a.NAME = 'temperature' AND b.NAME = 'dew_point' AND a.TIME = b.TIME
ORDER BY a.TIME`)
CHART(
	chartOption({
		title: { text: "temperature vs dew_point", left: "center" },
		legend: { data: ["temperature", "dew_point"], top: 30 },
		grid: { top: 70 },
		xAxis: { type: "time" },
		yAxis: {},
		series: [
			{ name: "temperature", type: "line", data: column(0).map(function(t, idx) { return [t, column(1)[idx]]; }) },
			{ name: "dew_point", type: "line", data: column(0).map(function(t, idx) { return [t, column(2)[idx]]; }) }
		]
	})
)
```


`chartOption({...})`'s body is not evaluated on the server as ordinary TQL
expressions; it is shipped as literal JS source and executed in the browser
against a generated `column(idx)` helper (`function column(idx) { return
_columns[idx]; }`). Server-only TQL helpers such as `param()`, `value()`, and
`key()` do not exist in that browser context and cause `ReferenceError: ... is
not defined` in the browser console (the chart silently fails to render). To
use an HTTP query parameter (for example `?n=<tag>`) in a chart title or
series name, do not call `param()` inside `chartOption({...})`. Instead, bind
it into the `SQL()` source as usual and also select it as a normal column so
`column(idx)[0]` can read the scalar value client-side:

```tql
SQL(`SELECT TIME, VALUE, NAME FROM (
  SELECT TIME, VALUE, NAME FROM EXAMPLE WHERE NAME = ? ORDER BY TIME DESC LIMIT 100
) ORDER BY TIME`,
    param('n') ?? 'temperature')
CHART(
	chartOption({
		title: { text: (column(2)[0] || 'unknown') + ' - recent 100', left: 'center' },
		xAxis: { type: 'time' },
		yAxis: {},
		series: [{ name: column(2)[0], type: 'line', data: column(0).map(function(t, idx) { return [t, column(1)[idx]]; }) }]
	})
)
```

For a database-independent chart smoke test, use `SCRIPT()` to generate a sine wave:

```tql
SCRIPT({
	for (x = 0; x < 360; x += 3.6) {
		$.yield(x, Math.sin(x / 180 * Math.PI));
	}
})
CHART(
	chartOption({
		xAxis: {
			type: "category",
			data: column(0)
		},
		yAxis: {},
		series: [
			{
				type: "line",
				data: column(1)
			}
		]
	})
)
```

The MCP server converts this chart envelope into self-contained HTML by loading `jsAssets` before `jsCodeAssets`, saves it under `<temp>/neo-mcp-<pid>/charts/<chartID>.html` by default, and serves the shared data root through a loopback HTTP server on an ephemeral `127.0.0.1` port. Use `-data-dir` to select another artifact root; charts are stored in its `charts/` subdirectory and linked through the local `/mcp/charts/<chartID>.html` service. The same loopback server proxies `/db/*`, `/web/*`, `/metrics/*`, and `/debug/*` to the configured machbase-neo endpoint with the configured MCP token. The MCP server must not place the API token in generated HTML or URLs.

The VS Code API provides the public `vscode.open` command for opening the
generated local HTML file. A future companion extension can watch the
configured data root's `charts/` directory and expose an
`neo-mcp.openLatestChart` command. The MCP
stdio process itself cannot directly invoke arbitrary VS Code commands, and a
dedicated Internal Browser command is not treated as a stable public API.

Use `fs_list` to discover server-side TQL files and `tql_run_file` to read and
execute a selected `.tql` file. The file path is an SSFS server path, not a
local workspace path.

## Opening a result URL

Both `tql_run` (chart HTML under `/mcp/charts/<chartID>.html`) and
`tql_file_link` (proxied `/db/tql/<path>.tql`) return a plain `http://127.0.0.1:<port>/...`
URL, not a rendering. This browser-first rule applies to any sink, not only
`CHART()`/HTML: a browser tab often renders a `BOX()`, `CSV()`, `JSON()`, or
`NDJSON()` result more readably than pasting it into the chat response,
especially for wide tables or large payloads (see "Box output" below). Do not
stop at printing the URL as a markdown link. Prefer this order when a
browser-capable agent surface is available:

1. Call the agent's internal/integrated browser tool (for example
   `open_browser_page`, or `navigate_page` when a suitable tab is already
   open) with the returned URL in the same turn, before writing the final
   response, so the user sees the rendered chart or page directly.
2. Only if no such browser tool is available or the call fails, fall back to
   presenting the URL as a clickable markdown link (`[Open chart](...)` or
   `[Open TQL](...)`) for the user to open manually.

This is a mandatory action, not an optional courtesy: whenever a
browser-capable surface is available, actually invoke the browser tool before
replying. Displaying the URL as a markdown link is only a fallback for when
no browser tool exists or the call failed; it is never a substitute for
calling the browser tool when one is available, and a markdown link should
not be the only thing shown in that case.

Both link forms are already token-free loopback URLs, so either fallback link
is safe to display as-is.

The loopback port is ephemeral and chosen per MCP process lifetime, so it
changes whenever the neo-mcp process restarts (for example after an MCP
server restart in the editor). Never reuse a previously seen `127.0.0.1:<port>`
URL from an earlier turn or an earlier browser tab without re-verifying it.
Before opening or navigating to a chart/TQL URL, re-fetch it with `tql_run` or
`tql_file_link` in the current turn and use the port that call just returned.
If an existing browser tab shows a stale port and now fails to load, re-run
`tql_file_link` for the same path and navigate the tab to the freshly returned
URL instead of assuming the old port is still valid.

## Server TQL files and browser links

For an iterative server-side workflow, use `fs_write` to create or replace a
`.tql` file, then call `tql_file_link`. The tool executes the file through the
same external reading API documented at
https://docs.machbase.com/neo/tql/reading.md and returns both the verification
result and a loopback browser URL in the form `/db/tql/<path>.tql`. The
loopback server adds the configured MCP token before forwarding the request to
machbase-neo.

Repeat `fs_write` and `tql_file_link` while refining the query or chart. This
path does not create an intermediate HTML file in neo-mcp. A `CHART()` sink
returns the server chart JSON envelope; CSV, JSON, NDJSON, Markdown, and HTML
sinks return their native reading-API output.

The returned URL contains no API token and does not require a separate browser
login. It is reachable while the neo-mcp process is running and uses the MCP
token owner's permissions. The loopback proxy only forwards the configured
machbase-neo HTTP prefixes; neo-mcp-local services live under `/mcp/*`.

### Box output

Use `BOX()` to render incoming records as an ASCII table. This is the preferred
sink when a user asks for a compact, human-readable table rather than a JSON or
CSV response.

```tql
SQL(`SELECT NAME, COUNT(*) AS record_count FROM EXAMPLE GROUP BY NAME`)
BOX()
```

`BOX()` returns `text/plain`, not HTML. Opening its `tql_run_file`/`tql_file_link`
URL in a browser just shows the raw ASCII table as plain text, not a rendered
widget; that is expected and is not a rendering failure. Either presentation
is acceptable for a `BOX()` (or `CSV()`/`JSON()`/`NDJSON()`) result: showing it
directly in the chat response is fine for a short result, and opening it in a
browser tab is often more readable for a longer table or payload. Use
judgment on result size rather than treating one presentation as mandatory.

`BOX()` accepts the same encoder options as `CSV()`/`JSON()`/`NDJSON()`, even
though its per-function doc is not yet detailed: `sqlTimeformat('DEFAULT')`/
`ansiTimeformat(...)` controls the `TIME` column format, and `tz('Local')` (or
an IANA zone) controls its time zone, renaming the column to `TIME(LOCAL)`.

```tql
SQL(`SELECT TIME, VALUE FROM EXAMPLE WHERE NAME = 'wind_speed' ORDER BY TIME DESC LIMIT 100`)
BOX(sqlTimeformat('DEFAULT'), tz('Local'))
```

For a server-side TQL file that should be opened in a browser, write it below
`/project`, execute it with `tql_run_file`, then call `tql_file_link`. The link
tool verifies the output and returns a token-free loopback URL.

### Table record counts

First call `db_list_tables` to obtain the visible table names. Machbase does
not accept a literal table name and `COUNT(*)` together in an aggregate query
unless the literal is grouped, so do not assume a query such as
`SELECT 'TABLE' AS table_name, COUNT(*) FROM TABLE` will work. A portable
approach is to union one `COUNT(*)` query per discovered table and prepend the
known table name in TQL using each SQL row's one-based `key()`.

```tql
SQL(`SELECT COUNT(*) AS record_count FROM COMPLEX
UNION ALL SELECT COUNT(*) AS record_count FROM EXAMPLE`)
PUSHVALUE(0, key() == 1 ? 'COMPLEX' : 'EXAMPLE', 'table_name')
BOX()
```

Keep the `UNION ALL` order and the `key()` conditions aligned. Execute the
completed script before writing a `.tql` file, and execute the written file
again with `tql_run_file` before presenting its `tql_file_link` URL.

## VS Code Copilot chart smoke test

Ask Copilot:

> Read `neo://manual/tql`, write the database-independent sine-wave TQL example above, execute it with `tql_run`, and show the clickable chart HTML file link.

Expected sequence:

1. `manual_read` for `neo://manual/tql`.
2. Write the `SCRIPT()` and `CHART()` script.
3. Call `tql_run` with the script.
4. Receive an interactive chart file link served from the configured data root's `charts/` directory.

## Function and signature reference

The neo-server TQL language service is the canonical metadata source. Its generated documentation contains function descriptions, signatures, argument slots, suggestions, examples, statement kinds, and source/sink role variants. Keep this manual aligned with that metadata when adding or changing TQL functions.

The most authoritative implementation assets are:

- `neo-server/mods/lsp/tql/docsrc/` for maintained per-function Markdown.
- `neo-server/mods/lsp/tql/docs_gen.go` for generated metadata consumed by the language service.
- `neo-server/mods/lsp/tql/service.go` for completion, hover, signature, and statement-kind behavior.

When a function is unknown, ask the user to inspect the available metadata rather than guessing its signature.
