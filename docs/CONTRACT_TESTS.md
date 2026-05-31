# Contract Tests

`graphgate test` executes persisted operations against an environment from `graphgate.yaml`.

```sh
graphgate test --env local
graphgate test --env staging --format json
graphgate test --env local --update
graphgate test --env local --tag smoke --exclude destructive
graphgate smoke --env local --suite local-readiness --evidence graphgate/reports/local-readiness.json
```

Secrets must come from environment variables referenced in config headers. GraphGate does not write request headers or token values to reports.

## Fixture Format

```json
{
  "name": "viewer smoke",
  "operation": "GetViewer",
  "requestOperationName": "Viewer",
  "includeQuery": false,
  "tags": ["smoke", "read", "positive"],
  "profile": "local-user",
  "headers": {
    "X-Trace-ID": "${GRAPHGATE_TRACE_ID}"
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

Fixtures can reference reusable config profiles:

```yaml
profiles:
  local-user:
    headers:
      X-User-ID: ${GRAPHGATE_USER_ID}
    variables:
      viewerID: ${GRAPHGATE_USER_ID}
```

Use fixture `profile: local-user` for one profile or `profiles:
["local-user", "review-context"]` for multiple profiles. Headers merge in this
order: environment, profiles, fixture. Variables merge profiles first, then
fixture variables.

## Tags And Suites

Add `tags` to fixtures and select them from the CLI:

```sh
graphgate test --tag smoke
graphgate smoke --tag read --exclude destructive
graphgate smoke --suite local-readiness
```

Named suites live under `tests.suites`:

```yaml
tests:
  suites:
    local-readiness:
      tags: [smoke, read]
      exclude: [destructive]
      maxLatencyMs: 2000
```

Multiple `--tag` values are combined as an AND selection. A fixture is excluded
when it has any listed `--exclude` tag. Suite filters compose with CLI filters.

Set `tests.requireOperationCoverage: true` in `graphgate.yaml` when a smoke gate
must prove that fixtures reference every validated operation at least once.
Additional coverage gates can require positive fixtures, required tags, no
unknown fixture operations, and no deprecated operation usage:

```yaml
tests:
  coverage:
    requirePositiveFixture: true
    requireTags: [smoke]
    forbidUnknownOperations: true
    forbidDeprecatedOperations: true
```

Assertions support:

- `exists`: `true` or `false`
- `type`: `string`, `number`, `boolean`, `object`, `array`, or `null`
- `equals`: exact JSON value comparison
- `minItems`: minimum length for an array value
- `nonEmpty`: string, array, object, or non-null value must be non-empty
- `matches`: regular expression for string values
- `contains`: array item, string substring, object key, or object value
- `allType`: every matched value or array item has the requested type
- `allNonNull`: every matched value or array item is non-null
- `min` and `max`: numeric bounds

Paths support dot notation, numeric array indexes, and wildcards:

```json
{ "path": "data.items[*].id", "matches": "^[A-Za-z0-9_-]+$" }
```

## Readiness

Environment readiness can fail a run before fixture execution:

```yaml
environments:
  local:
    endpoint: http://localhost:8080/graphql
    requiredEnv: [GRAPHGATE_TOKEN]
    healthCheck:
      path: /health
      expectedStatus: 200
      timeout: 2s
```

Use `--skip-readiness` for local debugging when the endpoint is intentionally
partial.

## Evidence

Use `--evidence` to persist a reproducible run bundle:

```sh
graphgate smoke --suite local-readiness --evidence graphgate/reports/local-readiness.json
```

The JSON bundle includes command metadata, config/schema/manifest hashes,
environment, endpoint, selected tags or suite, readiness results, per-fixture
status, duration, attempts, failure class, GraphQL error codes, assertion
results, and timing summary. `--evidence-format markdown` writes the markdown
test report, and `--evidence-format both` writes JSON plus a sibling markdown
file.

## Negative And Security Fixtures

Negative fixtures use the same format as positive fixtures, but should carry
tags such as `negative` and `security` and set the expected failure boundary:

```json
{
  "operation": "GetViewer",
  "tags": ["smoke", "negative", "security"],
  "expectedStatus": 200,
  "expectedErrorCodes": ["UNAUTHORIZED"],
  "expectedFailureClass": "graphql_error"
}
```

When the expected status and GraphQL error codes match, the fixture passes and
the evidence still records the expected failure class.

## Snapshot Modes

`graphgate test --update` writes the latest response into the fixture `snapshot`
field. Snapshot comparison defaults to `exact` for backward compatibility.
Configure `shape` for response-shape snapshots or `disabled` to skip snapshots:

```yaml
tests:
  snapshot:
    mode: shape
    ignorePaths: [data.viewer.generatedAt]
    redactPaths: [data.viewer.email]
```

Fixture-level `snapshotMode`, `snapshotIgnorePaths`, and `snapshotRedactPaths`
can narrow behavior for one fixture.

## Latency And Retry

Set `maxLatencyMs` at the test, suite, or fixture level to fail slow endpoint
responses. Retry policy is configured in `tests.retry` and defaults to retrying
only `network_error` and `timeout` failures when `maxAttempts` is greater than
one.
