# Config Reference

GraphGate reads `graphgate.yaml` by default. Use `--config path/to/graphgate.yaml` to point commands at another file.

## Example

```yaml
schema: ./schema.graphql
operations:
  - ./operations/**/*.graphql
manifest:
  format: graphgate
  output: ./graphgate.manifest.json
environments:
  local:
    endpoint: http://localhost:8080/graphql
    headers:
      Authorization: Bearer ${GRAPHGATE_TOKEN}
tests:
  fixtures: ./graphgate/fixtures/**/*.json
  requireOperationCoverage: false
reports:
  output: ./graphgate/reports
```

## Fields

### `schema`

Path to the GraphQL SDL schema file.

### `operations`

List of operation file paths or glob patterns. `**` recursive globs are supported for operation discovery.

### `manifest.format`

Manifest format identifier. GraphGate supports `graphgate`.

### `manifest.output`

Path where `graphgate manifest` writes the deterministic persisted-operation manifest.

### `environments`

Named GraphQL endpoints used by `graphgate test`.

Headers may reference environment variables, for example `${GRAPHGATE_TOKEN}`.
Fixture-level headers are merged after environment headers, which lets a smoke
suite switch between safe local users without duplicating environments.

### `tests.fixtures`

Glob pattern for fixture-based contract tests used by `graphgate test` and
`graphgate smoke`.

### `tests.requireOperationCoverage`

Optional boolean. When true, `graphgate test` and `graphgate smoke` fail before
running requests unless the fixture set references every validated operation at
least once. This is useful for strict endpoint smoke gates.

### `reports.output`

Default directory for generated reports.

## Exit Codes

| Code | Meaning |
| --- | --- |
| 0 | Success |
| 1 | Internal error |
| 2 | Invalid config |
| 3 | Invalid schema |
| 4 | Invalid operation |
| 5 | Manifest mismatch |
| 6 | Breaking schema change impacting operations |
| 7 | Contract test failure |
