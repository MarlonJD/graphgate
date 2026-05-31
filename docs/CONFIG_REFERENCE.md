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
    requiredEnv:
      - GRAPHGATE_TOKEN
    healthCheck:
      path: /health
      expectedStatus: 200
      timeout: 2s
    headers:
      Authorization: Bearer ${GRAPHGATE_TOKEN}
profiles:
  local-user:
    headers:
      X-User-ID: ${GRAPHGATE_USER_ID}
    variables:
      viewerID: ${GRAPHGATE_USER_ID}
tests:
  fixtures: ./graphgate/fixtures/**/*.json
  requireOperationCoverage: false
  suites:
    local-readiness:
      tags: [smoke]
      exclude: [destructive]
      maxLatencyMs: 2000
  coverage:
    requirePositiveFixture: true
    requireTags: [smoke]
    forbidUnknownOperations: true
    forbidDeprecatedOperations: true
  snapshot:
    mode: shape
    ignorePaths: [data.viewer.generatedAt]
    redactPaths: [data.viewer.email]
  retry:
    maxAttempts: 2
    backoff: 250ms
    retryableFailureClasses: [network_error, timeout]
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

Named GraphQL endpoints used by `graphgate test` and `graphgate smoke`.

Headers may reference environment variables, for example `${GRAPHGATE_TOKEN}`.
Profile headers and fixture-level headers are merged after environment headers,
which lets a smoke suite switch between auth contexts without duplicating
environments. Header values support environment-variable expansion.

### `environments.<name>.requiredEnv`

Optional list of environment variable names that must be set before endpoint
fixtures run. Missing values fail the run before network execution with
`readiness_failed`.

### `environments.<name>.healthCheck`

Optional endpoint readiness check. Use either `url` or `path`; `path` is resolved
against the environment endpoint host. `method` defaults to `GET`,
`expectedStatus` defaults to `200`, and `timeout` defaults to the command
timeout. Health-check headers merge environment headers first, then
health-check headers.

### `profiles`

Reusable header and variable profiles. Fixtures can reference one profile with
`profile` or multiple with `profiles`. Merge order is deterministic:

1. Environment headers.
2. Profile headers in fixture order.
3. Fixture headers.

Profile variables merge before fixture variables, so fixtures can override
profile defaults. String values in headers and variables are environment
expandable.

### `tests.fixtures`

Glob pattern for fixture-based contract tests used by `graphgate test` and
`graphgate smoke`.

### `tests.suites`

Named fixture selections for repeatable command lines. `tags` are inclusive and
all listed tags must be present on a fixture. `exclude` removes any fixture with
one of the listed tags. Command-line `--tag` and `--exclude` filters compose
with suite filters. `maxLatencyMs` sets a suite-level default latency threshold.

### `tests.requireOperationCoverage`

Optional boolean. When true, `graphgate test` and `graphgate smoke` fail before
running requests unless the fixture set references every validated operation at
least once. This is useful for strict endpoint smoke gates.

When a suite or tag filter is used, coverage gates apply to the selected fixture
set. Without filters, they apply to all validated operations.

### `tests.coverage`

Optional stricter coverage gates:

- `requirePositiveFixture`: require at least one non-negative fixture for each
  operation in the active coverage set.
- `requireTags`: require every selected fixture to include the listed tags.
- `forbidUnknownOperations`: fail before network execution when a fixture names
  an operation not found in the validated operation set.
- `forbidDeprecatedOperations`: fail when selected fixtures reference an
  operation that uses deprecated schema fields.

### `tests.snapshot`

Default snapshot behavior for fixtures that include a `snapshot` field or run
with `--update`.

- `mode`: `exact`, `shape`, or `disabled`. The default is `exact` for backward
  compatibility.
- `ignorePaths`: JSON paths to remove before comparison.
- `redactPaths`: JSON paths to replace with `[REDACTED]` before comparison or
  update.

Fixture-level `snapshotMode`, `snapshotIgnorePaths`, and `snapshotRedactPaths`
override or extend the defaults.

### `tests.retry`

Optional retry policy for endpoint tests:

- `maxAttempts`: total attempts per fixture. Defaults to `1`.
- `backoff`: duration between attempts, for example `250ms`.
- `retryableFailureClasses`: failure classes that may be retried. Defaults to
  `network_error` and `timeout`.

Assertion, GraphQL error, coverage, and snapshot failures are not retried unless
explicitly listed.

### `tests.maxLatencyMs`

Optional default latency threshold for every selected fixture. Suite-level
`maxLatencyMs` overrides this value, and fixture-level `maxLatencyMs` overrides
both.

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
