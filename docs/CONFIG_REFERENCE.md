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
reports:
  output: ./graphgate/reports
```

## Fields

### `schema`

Path to the GraphQL SDL schema file.

### `operations`

List of operation file paths or glob patterns. `**` recursive globs are supported for operation discovery.

### `manifest.format`

Manifest format identifier. M1 supports `graphgate`.

### `manifest.output`

Path where `graphgate manifest` writes the deterministic persisted-operation manifest.

### `environments`

Named GraphQL endpoints for later contract test runner milestones. M1 preserves the shape but does not execute operations.

Headers may reference environment variables, for example `${GRAPHGATE_TOKEN}`.

### `tests.fixtures`

Glob pattern for fixture-based contract tests used by `graphgate test`.

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
