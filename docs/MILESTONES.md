# Milestones

## M0: Product Foundation And Documentation

Status: complete.

- Product positioning and README.
- Product plan, milestone roadmap, and config reference.
- AGPL-3.0-or-later license.
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

Status: complete.

- `graphgate manifest --check` detects stale committed manifests.
- `graphgate diff --base <schema>` reports breaking changes and impacted operations.
- GitHub Actions examples for validation, manifest checks, and schema diff reports.
- Docker image scaffold for CI at `ghcr.io/marlonjd/graphgate`.
- Markdown report output suitable for PR comments and job summaries.

## M3: Contract Test Runner

Status: complete.

- `graphgate test --env local|staging` executes operation fixtures.
- Fixtures support variables, expected status, expected GraphQL error codes, JSON assertions, and snapshots.
- `graphgate test --update` refreshes fixture snapshots.
- Reports include passed/failed fixture counts without writing request headers or tokens.

## M4: Local Web UI

Status: complete.

- `graphgate ui` starts a local HTTP API and browser UI.
- Dashboard, operations, schema, manifest, diff, test runs, and reports screens are present.
- The same config and generated validation/manifest data are used by CLI and UI.
- No cloud login.

## M5: Distribution

Status: complete.

- GitHub Releases workflow builds macOS, Linux, and Windows archives.
- Checksums are generated and attested in release workflow.
- Homebrew formula with release checksums is included.
- Docker image workflow publishes `ghcr.io/marlonjd/graphgate`.
- Composite GitHub Action is included.
- macOS, Linux, Windows, Docker, and GitHub Action install docs are included.

## M6: EMSI Dogfood And REST Migration Mode

Status: complete.

- EMSI dogfood setup stays external and can live at `tools/graphgate/`.
- Example config validates `emsi_go_api/internal/graph/schema.graphqls`.
- Sample EMSI operations cover `cities`, `activeEmojis`, and `viewer`.
- REST-to-GraphQL parity diff workflow is documented with mapping and response examples.
