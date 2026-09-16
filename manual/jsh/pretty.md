# JSH `pretty` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/pretty.md

`pretty` formats values and tabular results for concise JSH stdout.

```js
const pretty = require('pretty');
```

## Table

`pretty.Table(config)` supports `box`, `csv`, `tsv`, `json`, `ndjson`, `html`,
and `md` formats. Useful options include `boxStyle`, `rownum`, `timeformat`,
`tz`, `precision`, `header`, `footer`, `nullValue`, and `stringEscape`.

Methods include `appendHeader`, `appendRow`, `appendRows`, `append`, `row`,
`render`, `close`, `resetRows`, and `pauseAndWait`.

```js
const tw = pretty.Table({ format: 'md', rownum: false });
tw.appendHeader(['NAME', 'VALUE']);
tw.append(['barn', 0.03135]);
tw.append(['furnace', 0.0207]);
console.println(tw.render());
```

Use `format: 'ndjson'` for machine-readable row streaming and `format: 'box'`
or `format: 'md'` for human-readable chat output. Keep tables bounded before
printing them into an LLM context.

## Formatting helpers

- `MakeRow(size)` creates an empty row array.
- `Bytes(value)` formats byte counts.
- `Ints(value)` formats integers with grouping separators.
- `Durations(nanoseconds)` formats durations in compact units.
- `Align` provides alignment constants.
- `isTerminal()`, `getTerminalSize()`, `pauseTerminal()`, and `parseTime()` are terminal helpers.

`Progress(options)` creates a progress writer. It is intended for interactive
terminal sessions; in non-interactive MCP execution, ensure timers are stopped
and the tracker reaches `isDone()` before exiting.
