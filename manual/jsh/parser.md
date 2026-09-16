# JSH `parser` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/parser.md

`parser` provides streaming CSV and NDJSON decoders for JSH applications.

```js
const parser = require('parser');
```

## CSV

`parser.csv(options)` or `new parser.CSVParser(options)` accepts `separator`,
`quote`, `escape`, `headers`, `skipLines`, `skipComments`, `strict`,
`mapHeaders`, `mapValues`, and `trimLeadingSpace`.

```js
const fs = require('fs');
const parser = require('parser');

fs.createReadStream('/work/sample.csv', { encoding: 'utf8' })
    .pipe(parser.csv({ headers: true, strict: true }))
    .on('data', (row) => console.println(row.name, row.age))
    .on('error', (err) => console.println('parse error:', err.message));
```

The parser emits `headers`, `data`, `error`, and `end` events. With
`headers: false`, fields are named `"0"`, `"1"`, etc. `bytesWritten` and
`bytesRead` expose progress.

## NDJSON

`parser.ndjson(options)` or `new parser.NDJSONParser(options)` parses one JSON
object per line. `strict` defaults to `true`; with `strict: false`, invalid
lines emit `warning` objects containing line, data, and error.

```js
fs.createReadStream('/work/events.ndjson', { encoding: 'utf8' })
    .pipe(parser.ndjson({ strict: false }))
    .on('data', (event) => console.println(event.id))
    .on('warning', (warning) => console.println('skipped:', warning.line));
```

Both parsers are stream transforms. Prefer them over loading large files with
`readFile()`; always attach error handling and close the owning stream.
