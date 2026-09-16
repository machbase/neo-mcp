# JSH ETL Application Pattern

Read `neo://manual/jsh/machcli` before using this pattern. The application runs
on the machbase-neo server through `jsh_exec`; a script written with `fs_write`
is visible under `/work`.

## Query, transform, insert

Use a bounded source query while developing, transform rows in JavaScript, and
write only the derived values required by the destination schema.

```js
const { Client } = require('machcli');
const db = new Client({ host: '127.0.0.1', port: 5656, user: 'sys', password: 'manager' });
let conn;
let rows;
try {
    conn = db.connect();
    rows = conn.query('SELECT NAME, VALUE FROM EXAMPLE LIMIT 1000');

    const totals = {};
    for (const row of rows) {
        const name = String(row.NAME);
        const value = Number(row.VALUE);
        totals[name] = (totals[name] || 0) + value;
    }

    conn.exec('CREATE TABLE IF NOT EXISTS EXAMPLE_SUMMARY (NAME VARCHAR(100), TOTAL DOUBLE)');
    for (const name of Object.keys(totals)) {
        conn.exec('INSERT INTO EXAMPLE_SUMMARY VALUES (?, ?)', name, totals[name]);
    }
    console.println('inserted summaries:', Object.keys(totals).length);
} catch (err) {
    console.println('ETL error:', err.message);
} finally {
    if (rows) rows.close();
    if (conn) conn.close();
    db.close();
}
```

For high-volume ingestion, prefer `conn.append(tableName)` with `append()`,
`flush()`, and `close()` rather than one `exec()` per row. Transactions are
available through `db.tx()` or `conn.tx()` only for regular tables created with
`CREATE TABLE`; tag/log tables do not support transactions.

## MCP sequence

```text
manual_read(neo://manual/jsh)
manual_read(neo://manual/jsh/machcli)
manual_read(neo://manual/jsh/etl)
fs_write(/etl_example.js, script)
jsh_exec(require('/work/etl_example.js'))
```

Keep stdout concise: report counts, rejected rows, and destination status rather
than printing every input record. Treat DDL, inserts, appends, and generated
files as real side effects that require the token owner's permissions and MCP
approval.
