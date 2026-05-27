# Milestones

## M0: Product Foundation And Documentation

Status: complete.

- Product positioning and README.
- Product plan, milestone roadmap, and config reference.
- MIT license.
- Repository hygiene and basic CI.

## M1: CLI Core

Status: complete.

- `graphgate init` creates starter config and directories.
- `graphgate validate` validates schema and operations.
- `graphgate manifest` writes deterministic persisted-operation manifests.
- `graphgate manifest --check` detects stale manifests.
- `graphgate report --format markdown|json` emits validation and manifest summaries.
- Stable exit codes for invalid config, invalid schema, invalid operation, and manifest mismatch.

## M2: CI/CD Gate

Status: planned.

- Schema diffing with impacted operation reporting.
- GitHub Actions example for PR checks.
- Docker image for CI.
- Markdown PR comment output.

## M3: Contract Test Runner

Status: planned.

- Fixture-based operation execution.
- Expected status, GraphQL error code, and JSON shape assertions.
- Snapshot update flow.
- Redacted test reports.

## M4: Local Web UI

Status: planned.

- `graphgate ui` local API and embedded React UI.
- Dashboard, operations, schema, manifest, diff, test runs, and reports screens.
- No cloud login.

## M5: Distribution

Status: planned.

- GitHub Releases with checksums.
- Homebrew formula.
- Docker image.
- GitHub Action.
- Windows executable download.

## M6: EMSI Dogfood And REST Migration Mode

Status: planned.

- EMSI dogfood setup from an external GraphGate repo.
- Example config for `emsi_go_api/internal/graph/schema.graphqls`.
- REST-to-GraphQL parity diff path.
