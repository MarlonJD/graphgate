# GraphGate Guide

GraphGate is a CLI-first GraphQL contract gate with a local browser UI. It is
designed to catch GraphQL schema, operation, manifest, diff, and contract-test
failures before they reach production.

GraphGate is not a runtime client library and it is not a hosted graph platform.
It is a release-safety tool that runs in local development, CI/CD, and migration
workflows.

## The Problem

GraphQL schemas are typed, but production breakages still happen when the typed
contract is not checked at release time. Common failure modes include:

- A backend removes or renames a field that a client operation still uses.
- A schema change is valid GraphQL but breaks an existing mobile or web client.
- Persisted-operation hashes change because operation text was reformatted or
  regenerated inconsistently.
- A manifest generated locally is not committed, so CI deploys stale persisted
  operation data.
- A resolver keeps the same schema shape but starts returning different error
  codes or response structures.
- REST-to-GraphQL migrations lack a simple repeatable parity check.

Exploration tools such as GraphiQL, Altair, and Apollo Explorer are useful for
humans writing and running queries. They are not intended to be a deterministic
CI gate for every committed operation and release artifact. GraphGate fills that
gap.

## What GraphGate Checks

GraphGate focuses on contract surfaces that can break deployments:

- Schema parsing: the configured SDL must parse successfully.
- Operation validation: every committed operation must validate against the
  schema.
- Operation identity: operation IDs are generated from normalized operation text
  with SHA-256.
- Manifest determinism: generated manifests are ordered and stable.
- Manifest freshness: `graphgate manifest --check` fails if the committed
  manifest does not match the generated one.
- Schema diff impact: `graphgate diff --base` reports breaking schema changes
  and the operations they affect.
- Contract tests: `graphgate test` runs fixture-backed GraphQL requests against
  a configured endpoint.
- Local inspection: `graphgate ui` serves a local browser UI for the same
  project data.

## Core Workflow

Start by creating a config:

```sh
graphgate init
```

Then add or confirm:

- A GraphQL SDL schema file.
- Operation files under `operations/**/*.graphql`.
- A persisted-operation manifest output path.
- Optional contract-test fixtures.
- Optional local or staging GraphQL endpoints.

Typical local workflow:

```sh
graphgate validate
graphgate manifest
graphgate report --format markdown
graphgate ui
```

Typical CI workflow:

```sh
graphgate validate
graphgate manifest --check
graphgate diff --base schema.base.graphql
```

Optional runtime contract workflow:

```sh
graphgate test --env staging
```

## Configuration Model

GraphGate reads `graphgate.yaml` by default:

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

Paths are resolved relative to the config file. Header values can reference
environment variables. Secrets should stay in the environment and should not be
committed to config, fixtures, or reports.

See [CONFIG_REFERENCE.md](CONFIG_REFERENCE.md) for all fields.

## Commands

### `graphgate init`

Creates a starter `graphgate.yaml` and local directories:

- `operations/`
- `graphgate/fixtures/`
- `graphgate/reports/`

Use this when adopting GraphGate in a new project.

### `graphgate validate`

Parses the schema and validates every configured operation file. This is the
lowest-cost CI gate and should run on every pull request.

It catches:

- Invalid schema syntax.
- Invalid GraphQL operation syntax.
- Unknown fields.
- Type mismatches.
- Duplicate operation names.
- Anonymous operations.

### `graphgate manifest`

Generates a deterministic persisted-operation manifest. Operation bodies are
normalized before hashing, and output ordering is stable.

Use:

```sh
graphgate manifest
```

to write the manifest, and:

```sh
graphgate manifest --check
```

to fail CI when the generated manifest differs from the committed manifest.

### `graphgate diff`

Compares the current schema with a base schema and reports breaking changes plus
impacted operations:

```sh
graphgate diff --base schema.base.graphql
```

Use this to catch release compatibility problems before merging backend schema
changes.

### `graphgate test`

Runs fixture-backed GraphQL requests against an environment from config:

```sh
graphgate test --env local
graphgate test --env staging
graphgate test --env staging --update
```

Fixtures can assert:

- Expected HTTP status.
- Expected GraphQL error codes.
- JSON path existence.
- JSON value type.
- Exact JSON values.
- Snapshot responses.

See [CONTRACT_TESTS.md](CONTRACT_TESTS.md).

### `graphgate report`

Renders validation and manifest state as markdown or JSON:

```sh
graphgate report --format markdown
graphgate report --format json
```

Markdown reports are suitable for pull request comments and GitHub Actions job
summaries.

### `graphgate ui`

Starts a local HTTP server and browser UI:

```sh
graphgate ui
```

The default URL is:

```text
http://localhost:4317
```

The UI is local-only and does not require a cloud login.

## CI/CD Usage

GraphGate is intended to be strict in CI. A failed contract check should block a
merge or deployment.

Recommended pull request checks:

```sh
graphgate validate --config graphgate.yaml
graphgate manifest --config graphgate.yaml --check
graphgate diff --config graphgate.yaml --base schema.base.graphql
```

Recommended runtime or staging checks:

```sh
graphgate test --config graphgate.yaml --env staging
```

Install options:

- GitHub Action: `MarlonJD/graphgate@v0.1.1`.
- Release binary install script: `scripts/install.sh`.
- Docker image: `ghcr.io/marlonjd/graphgate:v0.1.1`.
- Homebrew: `brew install marlonjd/tap/graphgate`.
- Source build: `go install github.com/MarlonJD/graphgate/cmd/graphgate@v0.1.1`.

See [CI.md](CI.md) and [DISTRIBUTION.md](DISTRIBUTION.md).

## Persisted Operations

Persisted operations need stable IDs. If operation text is reformatted or
generated inconsistently, clients and servers can disagree on hashes.

GraphGate normalizes operation text before hashing and writes a deterministic
manifest. This makes manifest drift visible in code review and enforceable in
CI.

Recommended policy:

- Commit operation files.
- Commit the generated manifest.
- Run `graphgate manifest --check` in CI.
- Treat manifest mismatch as a release-blocking failure.

## Schema Diff And Impact

Not every schema change is a release risk. Adding a field usually should not
block a release. Removing or changing a field that a committed operation uses
should block the release.

GraphGate compares schema versions and maps breaking changes to operation field
usage. The diff report is designed for CI output and pull request summaries.

Use it when:

- A backend schema changes.
- A mobile app depends on persisted operations.
- Frontend releases are decoupled from backend releases.
- You need to understand which operations are affected by a schema change.

## Contract Test Fixtures

Validation proves an operation is structurally valid. Contract tests prove an
endpoint still behaves as expected for representative inputs.

Fixture tests are useful for:

- Authentication-sensitive operations.
- Error-code compatibility.
- Minimal response shape guarantees.
- Staging smoke tests.
- REST-to-GraphQL migration checks.

Secrets must come from environment variables referenced in `graphgate.yaml`.
GraphGate does not write request headers or token values to reports.

## REST-To-GraphQL Migration

GraphGate includes a documented path for REST parity work:

- Capture the existing REST response.
- Add the equivalent GraphQL operation.
- Capture the GraphQL response.
- Map REST JSON paths to GraphQL JSON paths.
- Compare the mapped output.
- Move the GraphQL operation into validation and manifest checks.

See [REST_PARITY_MODE.md](REST_PARITY_MODE.md).

## Local UI

The local UI is intended for inspection, not as a required hosted service. It
helps developers see the same project state that CI sees:

- Dashboard state.
- Operation names and hashes.
- Schema type and field explorer.
- Manifest preview.
- Diff and test entry points.
- Report exports.

The UI runs from `graphgate ui` and serves local project data.

## Adoption Path

Adopt GraphGate in stages:

1. Add `graphgate.yaml`.
2. Extract or add operation files.
3. Run `graphgate validate` locally.
4. Commit the generated manifest.
5. Add `graphgate validate` and `graphgate manifest --check` to CI.
6. Add schema diff checks once a base schema is available.
7. Add contract-test fixtures for high-risk operations.
8. Use `graphgate ui` for local inspection and debugging.

This lets a team start with a small validation gate and increase coverage as the
GraphQL contract becomes more critical.

## When GraphGate Is A Good Fit

GraphGate is a strong fit when:

- GraphQL operations are committed in the repo.
- Mobile or frontend clients rely on persisted operations.
- Backend and client releases happen independently.
- A schema change can break an already-shipped client.
- CI needs deterministic exit codes and readable reports.
- The team wants local-first tooling without a hosted graph service.

GraphGate is less useful when:

- The project has no committed operation files.
- GraphQL is used only for manual exploration.
- A hosted graph registry already enforces every contract check the team needs.
- The team needs a query authoring IDE rather than a release gate.

## Exit Codes

GraphGate uses stable exit codes for CI:

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

## Related Documentation

- [CONFIG_REFERENCE.md](CONFIG_REFERENCE.md)
- [CI.md](CI.md)
- [CONTRACT_TESTS.md](CONTRACT_TESTS.md)
- [DISTRIBUTION.md](DISTRIBUTION.md)
- [UI.md](UI.md)
- [REST_PARITY_MODE.md](REST_PARITY_MODE.md)
