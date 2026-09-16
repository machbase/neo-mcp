# JSH `readline` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/readline.md

`readline` provides synchronous-looking interactive input backed by the native
JSH terminal reader. It is mainly useful for interactive applications, not
non-interactive MCP scripts.

```js
const { ReadLine } = require('readline');
const reader = new ReadLine({ prompt: () => 'input> ' });
const line = reader.readLine();
if (line instanceof Error) throw line;
console.println(line);
reader.close();
```

Options include:

- `history`: base filename stored below `$HOME/.config/.jsh/`
- `prompt(lineno)`: returns the prompt string
- `submitOnEnterWhen(lines, idx)`: controls multiline submission
- `autoInput`: predefined input for tests and automation

`readLine([options])` returns a string or an `Error`. `addHistory(line)` adds a
history entry and `close()` ends the reader; closing a pending read produces
EOF. `ReadLine` also exposes key constants such as `Enter`, `CtrlJ`, `Escape`,
and navigation keys.

For MCP one-shot execution, prefer script arguments or `fs_read` input over
interactive `readline`, otherwise a script can wait indefinitely for stdin.
