# Local Web UI

Run the local browser UI with:

```sh
graphgate ui
```

By default GraphGate listens on:

```text
http://localhost:4317
```

Use a custom address:

```sh
graphgate ui --addr 127.0.0.1:4320
```

The UI reads the same `graphgate.yaml` as the CLI and exposes:

- Dashboard
- Operations
- Schema explorer
- Manifest preview
- Diff entry point
- Test run entry point
- Reports

The UI is local-only and does not require a cloud login.
