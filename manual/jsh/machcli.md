# JSH `machcli` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/machcli.md

`machcli` is the Machbase database client for JSH applications. It runs in the Goja runtime and is loaded with `require('machcli')`.

## Connection

```js
const { Client } = require('machcli');
const db = new Client({
    host: '127.0.0.1',
    port: 5656,
    user: 'sys',
    password: 'manager',
    database: 'MACHBASEDB'
});
const conn = db.connect();
```

Configuration fields:

- `host`, `port`, `user`, `password`
- `alternativeHost`, `alternativePort`
- `database` or its alias `db` for a logical database

Always close result sets, connections, and clients in `finally` blocks. The
credentials in examples are development placeholders; do not print or embed
real tokens/passwords in generated scripts.

## Query

`conn.query(sql, ...params)` returns a `Rows` result. It supports positional
`?` parameters and, on supported versions, named parameters:

```js
let rows;
try {
    rows = conn.query(
        'SELECT NAME, TIME, VALUE FROM EXAMPLE WHERE NAME = :name LIMIT :limit',
        { name: 'barn', limit: 20 }
    );
    for (const row of rows) {
        console.println(row.NAME, row.TIME, row.VALUE);
    }
} finally {
    if (rows) rows.close();
}
```

Positional form:

```js
rows = conn.query('SELECT * FROM EXAMPLE LIMIT ?', 100);
```

A `Row` exposes each column as `row.COLUMN_NAME` and is also iterable. Use
column properties for simple scripts and `for (const {key, value} of row)` when
generic column processing is needed.

A `DATETIME`/`TIME` column value is not a JS `Date`; it is a Go `time.Time`
object bridged into the runtime (`typeof` is `'object'`, and it is not
`instanceof Date`). `new Date(row.TIME)` fails with `Invalid time value`.
Use its own methods instead: `row.TIME.string()` for the default
`YYYY-MM-DD HH:MM:SS +ZZZZ ZZZ` text, `row.TIME.format(layout)` for a custom
Go-style layout, or `row.TIME.unixNano()`/`unixMilli()` for an epoch number.

Useful result methods:

- `rows.next()` returns `{value, done}`
- `rows.isFetchable()` checks whether rows can be fetched
- `rows.message()` returns the query message
- `rows.close()` releases the result

`conn.queryRow(sql, ...params)` returns one row object and is useful for
single-value queries such as counts or aggregates.

## DDL and DML

`conn.exec(sql, ...params)` executes DDL/DML and returns an object containing
`rowsAffected` and `message`:

```js
const result = conn.exec(
    'INSERT INTO MY_TABLE VALUES (?, ?, ?)',
    'sensor-1', new Date(), 12.34
);
console.println('affected:', result.rowsAffected, 'message:', result.message);
```

`conn.explain(sql, ...params)` returns an execution plan string.

## Transactions

`db.tx(fn)` obtains a connection, commits when `fn` returns normally, and rolls
back when `fn` throws. `conn.tx(fn)` provides the same behavior on an existing
connection.

Transactions are supported only for regular tables created with `CREATE TABLE`.
They are not supported for log tables or tag tables; those operations can fail
with errors such as `MACHCLI-ERR-2362`.

```js
try {
    db.tx(function (tx) {
        tx.exec('INSERT INTO TX_SAMPLE VALUES (?, ?)', 1, 'committed');
    });
} catch (err) {
    console.println('transaction failed:', err.message);
}
```

## Bulk append

`conn.append(tableName)` creates an appender for bulk inserts. Use
`append(...)`, `flush()`, and `close()`:

```js
const appender = conn.append('EXAMPLE');
appender.append('generated', new Date(), 1.25);
appender.flush();
const result = appender.close();
console.println(result);
```

Use appenders for high-volume ingestion rather than issuing one `exec()` per
row. Confirm the target table schema before generating values.

## Analysis pattern

The recommended LLM-generated application pattern is bounded query, explicit
conversion, concise stdout, and deterministic cleanup:

```js
const { Client } = require('machcli');
const db = new Client({ host: '127.0.0.1', port: 5656, user: 'sys', password: 'manager' });
let conn;
let rows;
try {
    conn = db.connect();
    rows = conn.query('SELECT * FROM EXAMPLE LIMIT 1000');
    let count = 0;
    let sum = 0;
    for (const row of rows) {
        count += 1;
        sum += Number(row.VALUE);
    }
    console.println('rows:', count);
    console.println('value_average:', sum / count);
} catch (err) {
    console.println('analysis error:', err.message);
} finally {
    if (rows) rows.close();
    if (conn) conn.close();
    db.close();
}
```

For a server-side application written with `fs_write`, execute it with
`jsh_run_file` using the MCP `/project` path:

```text
jsh_run_file('/project/analysis.js')
```

The reserved JSH command supplies the server's configured `/work` mount.

## Metadata helpers

The module also provides helpers for database metadata and display formatting,
including `queryDatabaseId`, `queryTableType`, `stringTableType`,
`stringTableFlag`, `stringTableDescription`, `stringColumnType`,
`stringColumnFlag`, and `columnWidth`. Use these when a script needs to explain
schema or table metadata rather than guessing type names.

Keep output bounded. A large query result should be summarized or written to a
server-side artifact rather than printed in full into the LLM context.
