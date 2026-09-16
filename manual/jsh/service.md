# JSH `service` Module

Canonical reference: https://docs.machbase.com/neo/jsh/modules/service.md

`service` calls the machbase-neo service controller JSON-RPC API. It is
callback-based and may affect server services; do not use it for ordinary data
analysis unless the user explicitly requests service administration.

```js
const service = require('service');
const client = new service.Client({ timeout: 1000 });

client.status((err, services) => {
    if (err) {
        console.println('service error:', err.message);
        return;
    }
    console.println('services:', services.length);
});
```

`new service.Client({ controller, timeout })` uses the `SERVICE_CONTROLLER`
environment when `controller` is omitted. Main methods are:

- `call(method[, params], callback)`
- `status([name], callback)`
- `read(callback)`, `update(callback)`, `reload(callback)`
- `install(config, callback)`, `uninstall(name, callback)`
- `start(name, callback)`, `stop(name, callback)`
- `runtime.get(name, callback)`
- `details.get/add/update/set/delete(...)`

Connection failures, timeouts, and RPC errors arrive as the first callback
argument. `reload()` stops all currently running services before applying the
current configuration; use it only with explicit approval. Ensure callbacks
finish or timeout before a one-shot JSH script exits.
