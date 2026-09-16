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

The MCP server converts this chart envelope into self-contained HTML by loading `jsAssets` before `jsCodeAssets`, saves it under the workspace `.neo-mcp/charts/<chartID>.html`, and returns a workspace-relative Markdown link. In VS Code, click the link to open the chart in the editor or the default browser. The MCP server must not place the API token in the generated HTML.

Use `fs_list` to discover server-side TQL files and `tql_run_file` to read and
execute a selected `.tql` file. The file path is an SSFS server path, not a
local workspace path.

## VS Code Copilot chart smoke test

Ask Copilot:

> Read `neo://manual/tql`, write the database-independent sine-wave TQL example above, execute it with `tql_run`, and show the clickable chart HTML file link.

Expected sequence:

1. `manual_read` for `neo://manual/tql`.
2. Write the `SCRIPT()` and `CHART()` script.
3. Call `tql_run` with the script.
4. Receive an interactive chart file link under `.neo-mcp/charts/`.

## Function and signature reference

The neo-server TQL language service is the canonical metadata source. Its generated documentation contains function descriptions, signatures, argument slots, suggestions, examples, statement kinds, and source/sink role variants. Keep this manual aligned with that metadata when adding or changing TQL functions.

The most authoritative implementation assets are:

- `neo-server/mods/lsp/tql/docsrc/` for maintained per-function Markdown.
- `neo-server/mods/lsp/tql/docs_gen.go` for generated metadata consumed by the language service.
- `neo-server/mods/lsp/tql/service.go` for completion, hover, signature, and statement-kind behavior.

When a function is unknown, ask the user to inspect the available metadata rather than guessing its signature.
