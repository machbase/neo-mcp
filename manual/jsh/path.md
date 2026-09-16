# JSH `path` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/path.md

`require('path')` is POSIX-oriented by default: separator `/`, delimiter `:`.
Use `path.posix` or `path.win32` when behavior must be explicit.

## APIs

- `resolve(...segments)` resolves an absolute path from the current directory.
- `normalize(value)` resolves `.` and `..` segments.
- `isAbsolute(value)` checks whether a path is absolute.
- `join(...segments)` joins and normalizes segments.
- `relative(from, to)` computes a relative path.
- `dirname(value)`, `basename(value[, ext])`, `extname(value)` inspect names.
- `parse(value)` returns `root`, `dir`, `base`, `ext`, and `name`.
- `format(pathObject)` rebuilds a path from parsed fields.

```js
const path = require('path');

const file = path.join('/work', 'data', 'example.csv');
console.println(path.dirname(file));
console.println(path.basename(file));
console.println(path.extname(file));
console.println(JSON.stringify(path.parse(file)));
```

Path functions require string arguments and throw `TypeError` for invalid input.
Normalize and validate paths before using them with `fs`; do not concatenate
untrusted path fragments without checking the resulting mount boundary.
