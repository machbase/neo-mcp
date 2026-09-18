# Machbase SQL Guidance

Use the `db_query` tool for SQL sent to machbase-neo. For DBMS engine syntax
(SQL statements, functions, data types, error codes) beyond this quick guide,
read `neo://manual/sql/reference` — it links each DBMS SQL Reference chapter
and summarizes the manual's own AI-agent SQL generation checklist.

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

## LOG TABLE Arrival Time

Machbase LOG TABLEs provide the database-managed `_arrival_time` pseudo-column
for each inserted record. It is not included in `SELECT *`; select it explicitly
when the query result needs the record arrival time:

```sql
SELECT _arrival_time, *
FROM EXAMPLE_LOG
ORDER BY _arrival_time DESC
LIMIT 10
```

Table names that begin with `_` are hidden from the default `SHOW TABLES`
result. Use `SHOW TABLES WITH ALL` when discovering or checking the existence
of internal tables:

```sql
SHOW TABLES WITH ALL
```

Also reference `_arrival_time` explicitly when filtering, ordering, or applying
retention-related time ranges. Do not assume an application timestamp column is
equivalent to the database-managed arrival time.

## TAG TABLE lifecycle and TAG cache

Each `CREATE TAG TABLE` consumes TAG cache memory. TAG cache usage therefore
grows with the number of TAG TABLEs, even when a table was created only for a
test, demonstration, probe, or temporary workflow.

- Pair every temporary `CREATE TAG TABLE` with deterministic `DROP TABLE`
	cleanup. Do not rely on process exit or a later test to remove it.
- Close rows, appenders, and database connections that reference the table
	before dropping it.
- When cleanup runs as a different user, use the owner-qualified table name,
	for example `DROP TABLE demo_user.temp_tags`; an unqualified name can target
	the cleanup user's schema and leave the intended table allocated.
- Check and report cleanup errors instead of ignoring them. Leaked TAG TABLEs
	can make later table creation fail with
	`MACHCLI-ERR-1423, TAG cache exhausted`.
- In a test suite that shares one Machbase instance, register cleanup
	immediately after a successful create and ensure the table is dropped before
	its owner is dropped.

## Identifiers and Aggregate Aliases

Avoid SQL keywords as table names, column aliases, or CTE names. In
particular, do not use `ROWS` as an aggregate alias; `ROWS` is used by SQL
syntax and can produce a syntax error such as `MACHCLI-ERR-2010`.

Prefer descriptive aliases that cannot be confused with SQL clauses:

```sql
SELECT COUNT(*) AS row_count,
	   MIN(VALUE) AS min_value,
	   MAX(VALUE) AS max_value,
	   AVG(VALUE) AS average_value
FROM EXAMPLE
WHERE NAME = 'temperature'
```

Use names such as `row_count`, `record_count`, `total_value`, `min_value`,
`max_value`, and `average_value`. When a generated query fails near an alias,
rename the alias before changing the query logic. Do not quote a questionable
alias as the first workaround; prefer a safe unquoted identifier.

## Table Record Counts

Call `db_list_tables` before generating a query that references table names.
Use the returned identifiers as the source of truth, and count each table with
`COUNT(*) AS record_count`.

Do not mix a literal table label with `COUNT(*)` in the same aggregate
`SELECT` unless the database accepts the required grouping expression. For
example, `SELECT 'EXAMPLE' AS table_name, COUNT(*) FROM EXAMPLE` can fail with
`MACHCLI-ERR-2044` because the literal is not a group-by expression.

To collect counts from multiple known tables, use one aggregate query per
table with `UNION ALL`. This reliably produces one count per table; a caller
such as TQL can attach the corresponding table labels after the query.

```sql
SELECT COUNT(*) AS record_count FROM COMPLEX
UNION ALL
SELECT COUNT(*) AS record_count FROM EXAMPLE
```

Do not rely on a `DUAL` table for a single-row source: it may not exist in a
Machbase database. For a human-readable result through the query API, pass
`format=box`; for a TQL result, use the `BOX()` sink.

## Frequently Used SQL Functions

Canonical complete reference:
https://docs.machbase.com/dbms/reference/sql/functions/functions-full.md

Use the following functions for common analysis tasks. Check the complete
reference for argument types, edition/version availability, and function-specific
constraints before generating less common expressions.

### Aggregation and statistics

- `COUNT(*)`, `COUNT(column)`: row count or non-NULL value count
- `SUM(column)`, `AVG(column)`: total and average of numeric values
- `MIN(column)`, `MAX(column)`: extrema
- `MEDIAN(column)`, `MODE(column)`: exact median and most frequent numeric value
- `P05`, `P10`, `P90`, `P95`, `PERCENTILE_CONT`, `PERCENTILE_DISC`, `QUANTILE`: percentiles
- `STDDEV`, `STDDEV_POP`, `VARIANCE`, `VAR_POP`: dispersion
- `FIRST(sort_expr, return_expr)`, `LAST(sort_expr, return_expr)`: value at the first/last sort position
- `GROUP_CONCAT(column)`: concatenate grouped values; check edition constraints

Example:

```sql
SELECT NAME AS sensor_name,
	   COUNT(*) AS row_count,
	   AVG(VALUE) AS average_value,
	   MIN(VALUE) AS min_value,
	   MAX(VALUE) AS max_value,
	   P95(VALUE) AS p95_value
FROM EXAMPLE
GROUP BY NAME
ORDER BY sensor_name
```

### NULL, conditional, and conversion

- `NVL(value, replacement)`: replace NULL with a fallback
- `DECODE(value, search, result, ..., default)`: equality-based mapping
- `CAST(expression AS type)`: explicit type conversion; use a valid Machbase type
- `TO_NUMBER` / `TO_NUMBER_SAFE`: string-to-number conversion; the SAFE form returns NULL on invalid input
- `TO_DATE` / `TO_DATE_SAFE`: string-to-DATETIME conversion; the SAFE form returns NULL on invalid input
- `TO_CHAR(value[, format])`: format numbers and DATETIME values as strings

NULL input generally produces NULL output. Use SAFE conversion functions when
one malformed value should not abort the whole analysis.

### Datetime and time-series grouping

- `SYSDATE`, `NOW`: current server time
- `YEAR`, `MONTH`, `DAY`, `DAYOFWEEK`: datetime parts
- `FROM_TIMESTAMP`, `TO_TIMESTAMP`: nanosecond epoch conversion
- `FROM_UNIXTIME`, `UNIX_TIMESTAMP`: 32-bit Unix time conversion
- `DATE_TRUNC(field, datetime[, count])`: truncate to time boundaries
- `DATE_BIN(field, count, datetime[, origin])`: fixed-size time buckets
- `ADD_TIME(datetime, diff)`: add years/months/days and time components

For time-series queries, make timezone and output formatting explicit with the
`tz` and `timeformat` query options.

```sql
SELECT DATE_TRUNC('hour', TIME) AS hour_bucket,
	   AVG(VALUE) AS average_value
FROM EXAMPLE
GROUP BY hour_bucket
ORDER BY hour_bucket
```

### Strings and numeric helpers

- `LOWER`, `UPPER`: case conversion
- `LENGTH`: string length in bytes
- `SUBSTR`, `SUBSTRING_INDEX`: substring extraction
- `INSTR`: 1-based pattern position, or 0 when absent
- `LTRIM`, `RTRIM`, `LPAD`, `RPAD`: trimming and padding
- `REGEXP_LIKE`, `REGEXP_INSTR`, `REGEXP_SUBSTR`, `REGEXP_REPLACE`: regular-expression operations
- `ABS`, `ROUND`, `TRUNC`, `FLOOR`, `CEIL`, `MOD`: numeric operations
- `PI`, `POWER`/`POW`, `SQRT`, `LOG`, `LN`, `EXP`, `SIN`, `COS`, `TAN`: math functions

### JSON and analytic functions

- JSON extraction: `JSON_EXTRACT_STRING`, `JSON_EXTRACT_INTEGER`, `JSON_EXTRACT_DOUBLE`, `JSON_TYPEOF`
- JSON mutation: `JSON_SET`, `JSON_SET_JSON`, `JSON_REMOVE`
- JSON access operators: `json_column->'$.path'` and `json_column.member`
- Window functions require `OVER`, such as `LAG`, `LEAD`, and `NTILE`

When a function reports an argument-type error, inspect the table metadata and
cast explicitly rather than relying on implicit conversion. Function errors can
abort the statement; use SAFE variants where the source data is not clean.

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
