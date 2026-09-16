# Machbase SQL Guidance

Use the `db_query` tool for SQL sent to machbase-neo.

## Querying

- Prefer explicit column lists over `SELECT *`.
- Add a small `LIMIT` while exploring data.
- Use Machbase table and tag metadata before inventing column or tag names.
- Treat identifiers returned by metadata tools as authoritative.
- Keep exploratory queries read-only unless the configured DB user is intentionally a development user.

## Tool workflow

1. Call `db_list_tables` to discover visible tables.
2. Call `db_describe_table` with a table name. This runs the existing `DESC <table>` query API.
3. Call `db_list_tags` and `db_tag_stat` when working with tag tables.
4. Call `db_query` with a bounded query and inspect the returned columns, types, rows, and elapsed time.

## Result interpretation

The underlying HTTP API returns the existing machbase query response shape, including `success`, `reason`, `elapse`, and data such as `columns`, `types`, and `rows`. Preserve those fields when explaining results to the user.

## Security

The MCP server sends the configured API token as `Authorization: Bearer <token>`. Database privileges are controlled by the token owner; do not assume that a tool-level read-only hint replaces database permissions.
