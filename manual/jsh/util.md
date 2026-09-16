# JSH `util` Module Group

Canonical references:

- https://docs.machbase.com/neo/jsh/modules/util/
- https://docs.machbase.com/neo/jsh/modules/util/parseargs/
- https://docs.machbase.com/neo/jsh/modules/util/splitfields/
- https://docs.machbase.com/neo/jsh/modules/util/tail/

The `util` group provides small helpers used by JSH applications and built-in
commands. Load helpers by their direct module path.

```js
const parseArgs = require('util/parseArgs');
const splitFields = require('util/splitFields');
const tail = require('util/tail');
```

## `parseArgs`

`parseArgs(args, ...configs)` parses CLI-style arguments. Configuration supports
`options`, `positionals`, `command`, `strict`, `allowNegative`, `tokens`,
`usage`, and `description`. Option types include `boolean`, `string`, `integer`,
and `float`; camelCase option names map to kebab-case flags.

```js
const result = parseArgs(process.argv.slice(2), {
    options: {
        verbose: { type: 'boolean', short: 'v' },
        limit: { type: 'integer', default: 100 }
    },
    allowPositionals: true,
    positionals: ['file']
});
console.println(JSON.stringify(result.values));
console.println(JSON.stringify(result.namedPositionals));
```

The result can contain `values`, `positionals`, `namedPositionals`, `tokens`,
and a matched `command`. `parseArgs.formatHelp()` generates help text and
`parseArgs.toKebabCase()` converts option names. In strict mode, unknown options
and unexpected positionals throw errors.

## `splitFields`

`splitFields(text[, options])` splits on spaces, tabs, newlines, and carriage
returns while preserving quoted substrings. Quotes are removed and empty fields
are omitted. The current options argument is accepted but not used.

```js
const splitFields = require('util/splitFields');
console.println(JSON.stringify(splitFields('cmd "arg 1" "arg 2"')));
// ["cmd","arg 1","arg 2"]
```

This is useful for shell-like tokenization, not full command parsing. Unclosed
quotes consume the rest of the input; escape sequences inside quotes are not
interpreted specially.

## `tail`

`tail.create(path, { fromStart })` creates a polling file follower. Call
`poll()` periodically; it returns newly appended lines. Call `close()` when the
application exits. Rotation and truncation are detected during polling.

```js
const tail = require('util/tail');
const follower = tail.create('/work/app.log', { fromStart: false });
const lines = follower.poll();
for (const line of lines) console.println(line);
follower.close();
```

The `util/tail/sse` adapter emits Server-Sent Events through `writeHeaders`,
`poll`, `send`, `comment`, and `close`. It is intended for HTTP/CGI contexts;
ordinary MCP scripts should use `tail.create` and keep polling bounded.
