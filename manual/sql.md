# Machbase SQL Guidance

Use the `db_query` tool for SQL sent to machbase-neo.

## Querying

Database context is available through MCP Resources. Use `neo://machbase/session` for the
current logical database and user (`select current_database()` and `select current_user()`), and
`neo://machbase/databases` for
logical and mounted database names, `neo://machbase/tables` for current-database tables,
`neo://machbase/tables/{prefix}` for current-database prefix filtering, and
`neo://machbase/tables/{database}` or `neo://machbase/tables/{database}/{prefix}` for
another logical database. Use `neo://machbase/table/{table}` for the current table schema.

- Prefer explicit column lists over `SELECT *`.
- Add a small `LIMIT` while exploring data.
- Use Machbase table and tag metadata before inventing column or tag names.
- Treat identifiers returned by metadata tools as authoritative.
- Keep exploratory queries read-only unless the configured DB user is intentionally a development user.

## HTTP Query Output Options

The canonical HTTP query API is documented at
https://docs.machbase.com/neo/api-http/query.md. Its endpoint is
`/db/query`, and the query can be sent as GET parameters, POST JSON, or POST
form data. The default result format is `json`.

Supported result formats:

| `format` | Use |
| --- | --- |
| `json` | Structured response with `columns`, `types`, and `rows` |
| `csv` | CSV text, useful for streaming or file-oriented processing |
| `box` | Human-readable ASCII table for chat output |
| `ndjson` | One JSON object per row for streaming |

Useful options include:

- `timeformat`: `s`, `ms`, `us`, or `ns` (default `ns`)
- `tz`: `UTC`, `Local`, or an IANA time zone such as `Asia/Seoul`
- `binaryformat`: `hex`, `base64`, `bytes`, or `preview`
- `header=skip`: omit the header row for CSV/BOX output
- `precision`: control floating-point formatting; `-1` means no rounding
- `rownum=true`: include row numbers in tabular output
- `db`: execute against a named logical database
- `p`: positional JSON array or named JSON object for SQL bind parameters

JSON-only shaping options are mutually exclusive: `transpose=true` returns
column-oriented `cols`, `rowsFlatten=true` flattens row arrays, and
`rowsArray=true` returns an array of objects.

For a compact table in a chat response, the HTTP equivalent is:

```text
/db/query?q=select%20*%20from%20example%20limit%2010&format=box
```

The `db_query` MCP tool exposes these options directly and returns structured
JSON for `format=json`. For `format=box`, `format=csv`, and `format=ndjson`, it
returns the server's text response so the selected format is preserved.

## Tool workflow

1. Call `db_list_tables` to discover visible tables.
2. Call `db_describe_table` with a table name. This runs the existing `DESC <table>` query API.
3. Call `db_list_tags` and `db_tag_stat` when working with tag tables.
4. Call `db_query` with a bounded query and inspect the returned columns, types, rows, and elapsed time.

## Result interpretation

The underlying HTTP API returns the existing machbase query response shape, including `success`, `reason`, `elapse`, and data such as `columns`, `types`, and `rows`. Preserve those fields when explaining results to the user.

## Security

The MCP server sends the configured API token as `Authorization: Bearer <token>`. Database privileges are controlled by the token owner; do not assume that a tool-level read-only hint replaces database permissions.
