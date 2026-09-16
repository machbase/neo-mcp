# JSH `fs` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/fs.md

`fs` provides synchronous, Node.js-compatible file APIs. In server-side JSH applications, the server-configured SSFS mounts are visible through paths such as `/work`.

```js
const fs = require('fs');
const text = fs.readFile('/work/input.json', 'utf8');
fs.writeFile('/work/output.txt', text, 'utf8');
```

## Common APIs

- `readFile(path[, options])`, `writeFile(path, data[, options])`, `appendFile(path, data[, options])`
- `exists(path)`, `stat(path)`, `lstat(path)`, `access(path[, mode])`
- `readdir(path[, options])`, `mkdir(path[, options])`
- `rename(oldPath, newPath)`, `copyFile(src, dest[, flags])`, `cp(src, dest[, options])`
- `rm(path[, options])`, `rmdir(path[, options])`, `unlink(path)`
- `realpath(path)`, `readlink(path)`, `truncate(path[, len])`
- `createReadStream(path[, options])`, `createWriteStream(path[, options])`
- `open`, `close`, `read`, `write`, `fstat`, `fsync`

`readdir()` supports `withFileTypes` and `recursive`; the current runtime may
include `.` and `..` entries. `stat()` returns metadata and type predicates such
as `isFile()` and `isDirectory()`.

The module exports Node-compatible `Sync` aliases, including `readFileSync`,
`writeFileSync`, `readdirSync`, `mkdirSync`, and `statSync`. Access constants
include `F_OK`, `R_OK`, `W_OK`, and `X_OK`; open flags include `O_RDONLY`,
`O_WRONLY`, `O_RDWR`, `O_CREAT`, `O_TRUNC`, and `O_APPEND`.

## Streaming and cleanup

```js
const fs = require('fs');
const parser = require('parser');

fs.createReadStream('/work/input.csv', { encoding: 'utf8' })
    .pipe(parser.csv())
    .on('data', (row) => console.println(row.NAME));
```

Close descriptors and streams when the application owns them. File writes,
renames, deletes, permissions, and symlinks are real server-side side effects.
Use temporary paths plus rename for safer replacement and clean up temporary
files in error paths.
