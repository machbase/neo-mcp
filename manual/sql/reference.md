# Machbase DBMS LLM Reference

Canonical manual: https://docs.machbase.com/dbms/index.md

## How to use the manual

Use the DBMS manual chapters as a decision tree rather than treating all SQL as generic SQL:

- Getting Started: connection checks and basic commands.
- Core Concepts: table types, time model, ROLLUP, and retention.
- Table Design: choose TAG, LOG, TRANSACTION, LOOKUP, or VOLATILE according to workload.
- TAG tables and ROLLUP: tag schema, ingestion, time-range queries, rollup design and rebuild.
- Development and Integration: client APIs and application integration.
- Security and Access Control: accounts, privileges, AUTH KEY, and access boundaries.
- Reference: exact SQL syntax, functions, configuration, and system catalogs.

## LLM query rules

- Inspect the actual table metadata before generating a query.
- Prefer Machbase-supported syntax from the Reference chapter over generic SQL assumptions.
- Use time predicates and `LIMIT` for exploratory time-series queries.
- Never infer privileges from a successful connection; the API token owner determines accessible objects and operations.

The full manual is intentionally referenced by URL here. Add focused, stable guidance to `neo-mcp/manual/` when a repeated agent failure or a Machbase-specific workflow needs a durable correction.

## DBMS manual's own AI Agent Reference (chapter 16.8)

The DBMS manual publishes a chapter written specifically for LLM/agent consumers:
https://docs.machbase.com/dbms/reference/ai-agent-reference/. Its response
sequence (16.8.1 Agent Guide) applies directly to SQL generation for this
server:

1. Identify the product version, edition, table type, SDK, and target object.
2. Check Machbase-specific meanings via [terminology-disambiguation](https://docs.machbase.com/dbms/reference/ai-agent-reference/terminology-disambiguation/).
3. Check the [support-matrix](https://docs.machbase.com/dbms/reference/ai-agent-reference/support-matrix/) and [constraints-index](https://docs.machbase.com/dbms/reference/ai-agent-reference/constraints-index/).
4. Verify the actual syntax/API on the canonical reference page (tables below).
5. Include prerequisites, execution, result checks, and cleanup in examples.
6. For uncertain facts, state the version/command needed to verify them rather than guessing.

Before generating SQL specifically (16.8.8 sql-generation-rules):

- Check the server version and edition; check the target database, owner, table type, and `DESC` output (`db_describe_table`/`db_list_tables` here).
- Verify the statement form in the [SQL Syntax Dictionary](https://docs.machbase.com/dbms/reference/sql/syntax/) and argument/return types in the [Function Dictionary](https://docs.machbase.com/dbms/reference/sql/functions/).
- Check edition/table-type constraints in [Support Scope](https://docs.machbase.com/dbms/reference/support-scope-constraints/).
- Do not assume another DBMS's keywords, functions, hints, or transaction behavior apply here.
- Do not replace identifiers with parameter markers (bind params are for values, not table/column names).
- For time/range `DELETE`/`UPDATE`, run a `SELECT` first to inspect the target rows.
- Specify `ORDER BY` when result order matters; include result checks and cleanup in modification examples.
- On error, use the actual error text with [Troubleshooting](https://docs.machbase.com/dbms/troubleshooting/) rather than arbitrarily changing syntax.
- Prefer canonical `docs.machbase.com` URLs as evidence in explanations over internal source paths.
- Do not propose deleting data, restarting the server, terminating sessions, changing settings, or recovery steps without confirming the user's target and authorization scope.

## SQL Reference (16.1) topic links

Use these canonical pages instead of guessing syntax. Each accepts the DBMS's
time-series extensions (TAG `SERIES BY`, relative time literals, `ROLLUP`) in
addition to ANSI SQL.

| Topic | Link |
| --- | --- |
| SQL Reference overview | https://docs.machbase.com/dbms/reference/sql/ |
| SQL Syntax Dictionary (index) | https://docs.machbase.com/dbms/reference/sql/syntax/ |
| SELECT | https://docs.machbase.com/dbms/reference/sql/syntax/select-syntax/ |
| WITH / CTE | https://docs.machbase.com/dbms/reference/sql/syntax/cte-syntax/ |
| Named bind parameter | https://docs.machbase.com/dbms/reference/sql/syntax/named-bind-parameter-syntax/ |
| SELECT hint (incl. SAMPLING) | https://docs.machbase.com/dbms/reference/sql/syntax/select-hint-syntax/ |
| SEARCH / ESEARCH / REGEXP | https://docs.machbase.com/dbms/reference/sql/syntax/search-esearch-regexp-syntax/ |
| Set operator (`UNION ALL` only; no UNION/INTERSECT/EXCEPT) | https://docs.machbase.com/dbms/reference/sql/syntax/set-operator-syntax/ |
| PIVOT | https://docs.machbase.com/dbms/reference/sql/syntax/pivot-syntax/ |
| Window function / OVER | https://docs.machbase.com/dbms/reference/sql/syntax/window-function-over-syntax/ |
| SERIES BY (TAG tables) | https://docs.machbase.com/dbms/reference/sql/syntax/series-syntax/ |
| SAVE DATA INTO | https://docs.machbase.com/dbms/reference/sql/syntax/save-data-into-syntax/ |
| DDL (CREATE/ALTER/DROP) | https://docs.machbase.com/dbms/reference/sql/syntax/ddl-syntax/ |
| DML (INSERT/UPDATE/DELETE, incl. TAG/LOOKUP predicate forms) | https://docs.machbase.com/dbms/reference/sql/syntax/dml-syntax/ |
| LOAD DATA INFILE | https://docs.machbase.com/dbms/reference/sql/syntax/load-data-infile-syntax/ |
| VIEW | https://docs.machbase.com/dbms/reference/sql/syntax/view-syntax/ |
| INDEX | https://docs.machbase.com/dbms/reference/sql/syntax/index-syntax/ |
| RETENTION | https://docs.machbase.com/dbms/reference/sql/syntax/retention-syntax/ |
| BACKUP / RESTORE / MOUNT | https://docs.machbase.com/dbms/reference/sql/syntax/backup-restore-mount-syntax/ |
| ROLLUP / ROLLUP_REBUILD | https://docs.machbase.com/dbms/reference/sql/syntax/rollup-syntax/ |
| USER/AUTH | https://docs.machbase.com/dbms/reference/sql/syntax/user-auth-syntax/ |
| SYSTEM/SESSION/ALTER SYSTEM | https://docs.machbase.com/dbms/reference/sql/syntax/system-session-alter-syntax/ |
| DATABASE (multi-database) | https://docs.machbase.com/dbms/reference/sql/syntax/database-syntax/ |
| AUTO_INCREMENT | https://docs.machbase.com/dbms/reference/sql/syntax/auto-increment-syntax/ |
| Relative time expressions (e.g. `now - 1h`) | https://docs.machbase.com/dbms/reference/sql/relative-time/ |
| ROWID | https://docs.machbase.com/dbms/reference/sql/rowid/ |
| Data Type Dictionary | https://docs.machbase.com/dbms/reference/sql/types/ |
| Function Dictionary (index) | https://docs.machbase.com/dbms/reference/sql/functions/ |
| Aggregate functions | https://docs.machbase.com/dbms/reference/sql/functions/aggregation/ |
| Window/series functions | https://docs.machbase.com/dbms/reference/sql/functions/series/ |
| Regular expression functions | https://docs.machbase.com/dbms/reference/sql/functions/regex/ |
| JSON functions and dot notation | https://docs.machbase.com/dbms/reference/sql/functions/operators-json/ |
| Date/time functions | https://docs.machbase.com/dbms/reference/sql/functions/datetime/ |
| Complete function reference | https://docs.machbase.com/dbms/reference/sql/functions/functions-full/ |

## Machbase-specific syntax cheat sheet

The topic links above are easy to skip when a request "looks like" ordinary
SQL, but these items are Machbase extensions or restrictions that a
generic-SQL assumption gets wrong. Prefer these inline forms directly; open
the linked page only for edge cases (editions, precision limits, DST/origin
rules).

**Relative time literals** (suffix on `now`/`sysdate` or any DATETIME value;
requires Machbase 8.0.50+): `ns`, `us`, `ms`, `s`, `m`, `h`, `d`, `w`. No
month/year suffix — use `ADD_TIME()` for calendar months/years.

```sql
SELECT * FROM sensor_tag WHERE time > now - 1h;
SELECT * FROM maintenance_plan WHERE planned_at < now + 2d6h15m;
```

**Named bind parameters**: `:name` (letters/digits/underscore, must start
with a letter/`_`/`$`). One marker style per statement — do not mix `?` and
`:name` in the same statement via name-based APIs. Parameters can only take
the place of a value/expression, never an identifier (table/column name) or
`ORDER BY` direction.

```sql
SELECT ID, NAME FROM SENSOR_DATA
 WHERE CREATED_AT >= :from_time AND CREATED_AT < :to_time
 LIMIT :row_count;
```

**Set operator**: only `UNION ALL` is supported — no `UNION`, `INTERSECT`, or
`EXCEPT`. Result order is not guaranteed; wrap in an inline view and add
`ORDER BY` outside it if order matters.

```sql
SELECT * FROM (
    SELECT id, name, time FROM log_a
    UNION ALL
    SELECT id, name, time FROM log_b
) ORDER BY time DESC;
```

**PIVOT**: requires an inline view (subquery); `IN (...)` values must be
compile-time literals, not dynamic.

```sql
SELECT * FROM (
    SELECT name, time, value FROM sensor_tag
     WHERE time BETWEEN TO_DATE('2024-01-01 00:00:00','YYYY-MM-DD HH24:MI:SS')
                     AND TO_DATE('2024-01-01 01:00:00','YYYY-MM-DD HH24:MI:SS')
) PIVOT (AVG(value) FOR name IN ('sensor-01', 'sensor-02'));
```

**SERIES BY** (TAG tables): extracts contiguous runs of rows matching a
condition from an ordered result; `SERIESNUM()` labels each contiguous run.
Without `ORDER BY`, rows are ordered by `_ARRIVAL_TIME`.

```sql
SELECT time, value, SERIESNUM() AS series_id
  FROM tag WHERE name = 'PRESSURE-01'
 ORDER BY time
 SERIES BY value > 100.0;
```

**ROLLUP query function** `rollup(time_unit, period, basetime_column[, origin])`
reads from a pre-created `ROLLUP` object, not a plain `GROUP BY` truncation;
if no matching ROLLUP candidate exists the query fails rather than silently
scanning raw rows. `period` must be a literal integer, not a bind parameter.
`STDDEV`/`VARIANCE` are not directly selectable in a rollup query (`ERR-02816`)
— compute them outside an inline view from `SUM`, `COUNT`, `SUMSQ` instead.

```sql
SELECT name, rollup('min', 1, time) AS bucket,
       COUNT(value), SUM(value), MIN(value), MAX(value), AVG(value)
  FROM ch6_query
 WHERE time >= TO_DATE('2026-01-01 00:00:00','YYYY-MM-DD HH24:MI:SS')
 GROUP BY name, bucket ORDER BY name, bucket;
```

## Other DBMS reference chapters (16.2-16.8)

| Topic | Link |
| --- | --- |
| Configuration property dictionary | https://docs.machbase.com/dbms/reference/configuration/configuration/ |
| System catalog / metadata tables (`M$...`) | https://docs.machbase.com/dbms/reference/system-catalog/meta/ |
| Virtual tables (`V$...`) | https://docs.machbase.com/dbms/reference/system-catalog/virtual/ |
| Command-line tools (machsql, machadmin, machloader, csvimport/export, tagmetaimport) | https://docs.machbase.com/dbms/reference/command-line-tools/ |
| Support scope by edition/table type/privilege | https://docs.machbase.com/dbms/reference/support-scope-constraints/ |
| Error code dictionary (look up `MACHCLI-ERR-*` / server errors) | https://docs.machbase.com/dbms/reference/error-codes/ |
| AI Agent Reference index | https://docs.machbase.com/dbms/reference/ai-agent-reference/ |
| Machine-readable concise index | https://docs.machbase.com/llms.txt |
| Machine-readable full text | https://docs.machbase.com/llms-full.txt |

When a query fails or a feature's availability is unclear, look up the exact
error text in the Error Code Dictionary and the relevant Support Scope page
before changing the query by trial and error.

