# Contract Tests

`graphgate test` executes persisted operations against an environment from `graphgate.yaml`.

```sh
graphgate test --env local
graphgate test --env staging --format json
graphgate test --env local --update
graphgate smoke --env local
```

Secrets must come from environment variables referenced in config headers. GraphGate does not write request headers or token values to reports.

## Fixture Format

```json
{
  "name": "viewer smoke",
  "operation": "GetViewer",
  "requestOperationName": "Viewer",
  "includeQuery": false,
  "headers": {
    "X-User-ID": "42"
  },
  "variables": {},
  "expectedStatus": 200,
  "expectedErrorCodes": [],
  "assertions": [
    { "path": "data.viewer.friends", "type": "array", "minItems": 1 },
    { "path": "data.viewer.name", "exists": true }
  ]
}
```

`operation` or `operationName` identifies the GraphGate operation file used for
schema validation and, by default, the HTTP `operationName`. Set
`requestOperationName` when the runtime persisted operation name differs from
the GraphGate sample name. Set `includeQuery` to `false` to send only
`operationName` and variables to a persisted-only GraphQL endpoint.

Fixture `headers` are merged after environment headers and may reference
environment variables. String values in `variables` also support environment
variable expansion.

Set `tests.requireOperationCoverage: true` in `graphgate.yaml` when a smoke gate
must prove that fixtures reference every validated operation at least once.

Assertions support:

- `exists`: `true` or `false`
- `type`: `string`, `number`, `boolean`, `object`, `array`, or `null`
- `equals`: exact JSON value comparison
- `minItems`: minimum length for an array value

`graphgate test --update` writes the latest response into the fixture `snapshot` field.
