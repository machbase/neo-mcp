# JSH `process` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/process.md

`process` exposes JSH runtime state, arguments, environment, signals, output
streams, and process execution helpers.

## Runtime and arguments

```js
const process = require('process');
console.println(process.argv);
console.println(process.cwd());
console.println(process.platform, process.arch, process.pid);
console.println(process.env.get('HOME'));
```

Important APIs include `argv`, `cwd()`, `chdir(path)`, `env.get(name)`,
`env.set(name, value)`, `expand(value)`, `pid`, `ppid`, `platform`, `arch`,
`version`, `versions`, `uptime()`, and `now()`.

`argv[0]` is the JSH executable, `argv[1]` is the script path, and `argv[2:]`
contains script arguments.

## Output and execution

Use `console.println` for concise line output and `process.stdout`/
`process.stderr` for stream-oriented output. `process.which(command)` resolves a
JSH command, while `process.exec(command, ...args)` executes a command file and
`process.execString(source, ...args)` executes source text.

`process.exit(code)` ends the process. Do not use it for ordinary error handling
when a cleanup path is still needed.

## Cleanup and cancellation

`process.addShutdownHook(callback)` is useful for best-effort cleanup, but it is
not guaranteed for forceful termination such as SIGKILL. Use explicit `finally`
cleanup for database connections, rows, files, and transactions. Signal handlers
can be registered with `process.on('SIGTERM', handler)` and `process.on('SIGINT', handler)`.

`process.nextTick(callback, ...args)` schedules work on the next event-loop turn.
`process.hrtime()` is useful for measuring application phases.

`process.kill(pid[, signal])` sends a real OS signal and must be treated as a
high-impact side effect. Do not generate calls to arbitrary PIDs without an
explicit user request.
