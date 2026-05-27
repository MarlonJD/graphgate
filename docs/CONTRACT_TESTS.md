# Contract Tests

`graphgate test` executes persisted operations against an environment from `graphgate.yaml`.

```sh
graphgate test --env local
graphgate test --env staging --format json
graphgate test --env local --update
```

Secrets must come from environment variables referenced in config headers. GraphGate does not write request headers or token values to reports.

## Fixture Format

```json
{
  "name": "viewer smoke",
  "operation": "GetViewer",
  "variables": {},
  "expectedStatus": 200,
  "expectedErrorCodes": [],
  "assertions": [
    { "path": "data.viewer.id", "type": "string" },
    { "path": "data.viewer.name", "exists": true }
  ]
}
```

Assertions support:

- `exists`: `true` or `false`
- `type`: `string`, `number`, `boolean`, `object`, `array`, or `null`
- `equals`: exact JSON value comparison

`graphgate test --update` writes the latest response into the fixture `snapshot` field.
