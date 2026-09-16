# JSH `os` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/os.md

`os` provides host and runtime information. It is useful for diagnostics,
resource-aware summaries, and portable scripts.

```js
const os = require('os');
console.println('platform:', os.platform(), os.arch());
console.println('hostname:', os.hostname());
console.println('temp:', os.tmpdir());
console.println('cpus:', os.cpuCounts(true));
```

Common APIs:

- `arch()`, `platform()`, `type()`, `release()`, `hostname()`, `homedir()`
- `tmpdir()`, `endianness()`, `EOL`
- `totalmem()`, `freemem()`, `uptime()`, `bootTime()`, `loadavg()`
- `cpus()`, `cpuCounts(logical)`, `cpuPercent(intervalSec, perCPU)`
- `networkInterfaces()`, `hostInfo()`, `userInfo([options])`
- `diskPartitions([all])`, `diskUsage(path)`, `diskIOCounters([names])`
- `netProtoCounters([proto])`

`os.constants.signals` provides canonical signal numbers for use with
`process.kill`; `os.constants.priority` provides process priority values. Host
and network information can be sensitive, so print only the fields required by
the analysis request.
