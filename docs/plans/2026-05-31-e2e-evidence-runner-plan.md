# GraphGate End-To-End Evidence Runner Plan

Date: 2026-05-31

Owner subtree: `tools/graphgate`

## Goal

Evolve GraphGate from a schema/operation contract gate plus basic smoke runner
into a reusable, project-agnostic GraphQL endpoint evidence runner. The package
should help teams repeatedly prove that promoted operations are valid, covered
by fixtures, executable against real endpoints, and documented with portable
evidence artifacts.

## Assumptions

- GraphGate must remain generic and must not embed EMSI operation names, user
  IDs, seed data, endpoint assumptions, or milestone terminology.
- Existing commands remain compatible: `validate`, `manifest`, `report`,
  `test`, and `smoke` keep their current behavior unless new options are used.
- Fixture JSON remains the primary low-friction test definition format.
- Runtime network tests are opt-in and should produce actionable reports even
  when the endpoint is unavailable.

## Scope

Implement the ten features below in small, reviewable slices:

1. Tags and suites for selecting fixture subsets.
2. Reusable auth/header profiles.
3. Required environment variables and endpoint readiness checks.
4. Evidence bundle output with reproducible run metadata.
5. Expanded fixture coverage gates.
6. Snapshot modes with redaction and ignored paths.
7. Stronger JSON path and array assertions.
8. Negative and security fixture conventions.
9. Latency thresholds and timing summaries.
10. Retry policy with failure classification.

## Non-Goals

- Do not build a hosted registry, SaaS dashboard, or GraphQL router.
- Do not replace Apollo Rover, GraphQL Inspector, Hive, GraphQL Codegen, or
  Schemathesis. Prefer interop/export paths when those tools are better.
- Do not add project-specific fixture data or application semantics to
  `tools/graphgate`.
- Do not make network smoke tests run implicitly from `validate` or `manifest`.

## Phase 1: Selection And Reuse

Add fixture metadata and selection:

- Add optional `tags: []` to fixtures.
- Add CLI filters:
  - `graphgate test --tag smoke`
  - `graphgate smoke --tag m9c --exclude destructive`
  - `graphgate smoke --suite local-readiness`
- Add `tests.suites` to config so common tag combinations can be named.
- Add package tests for include/exclude behavior, unknown suites, and empty
  selected fixture sets.

Add reusable profiles:

- Add `profiles` to config with headers and optional variables.
- Add fixture-level `profile` or `profiles`.
- Merge order should be deterministic: environment headers, profile headers,
  fixture headers.
- Keep all profile values env-expandable.
- Add docs showing auth/header reuse without project-specific values.

Verification:

- `go test ./...`
- Fixture selection unit tests.
- CLI tests for `--tag`, `--exclude`, and `--suite`.

## Phase 2: Readiness And Evidence Artifacts

Add readiness checks:

- Add `environments.<name>.requiredEnv`.
- Add optional `environments.<name>.healthCheck` with method, path or URL,
  expected status, timeout, and optional headers.
- Add `graphgate smoke --skip-readiness` for local debugging.
- Fail before fixture execution when required env vars are missing or readiness
  fails, with a distinct classification.

Add evidence bundles:

- Add `--evidence <path>` to `test` and `smoke`.
- Write JSON evidence with:
  - run timestamp
  - command
  - GraphGate version or commit when available
  - config path and hash
  - schema hash
  - manifest hash
  - endpoint name and URL
  - selected tags/suite
  - per-fixture status, timing, request operation, GraphQL errors, assertions,
    and failure classification
- Add `--evidence-format json|markdown|both` if needed; default can be JSON
  while the existing stdout report remains unchanged.
- Redact configured secret headers and never persist request header values by
  default.

Verification:

- `go test ./...`
- Golden tests for evidence JSON shape.
- Tests proving secrets are redacted.

## Phase 3: Coverage Gates And Assertions

Expand coverage gates:

- Keep `tests.requireOperationCoverage`.
- Add optional gates:
  - `tests.coverage.requirePositiveFixture`
  - `tests.coverage.requireTags`
  - `tests.coverage.forbidUnknownOperations`
  - `tests.coverage.forbidDeprecatedOperations`
- Report missing coverage before network execution.

Improve assertions:

- Add JSONPath-like array support without introducing a large dependency unless
  the dependency is clearly maintained and small enough.
- Support:
  - wildcard paths
  - `nonEmpty`
  - `matches`
  - `contains`
  - `allType`
  - `allNonNull`
  - numeric min/max
- Preserve current dot-path assertions as the stable baseline.
- Add assertion failure messages that identify the path and expected condition.

Verification:

- `go test ./...`
- Unit tests for every new assertion operator.
- Backward compatibility tests for existing assertion fixtures.

## Phase 4: Snapshot, Negative, Timing, And Retry Semantics

Add snapshot modes:

- `snapshot.mode: exact|shape|disabled`
- `snapshot.ignorePaths`
- `snapshot.redactPaths`
- `snapshot.update: true` remains CLI-controlled by `--update`.
- Keep exact snapshots opt-in to avoid brittle endpoint tests by default.

Add negative/security fixture conventions:

- Document expected failure fixtures with:
  - `expectedStatus`
  - `expectedErrorCodes`
  - `expectedFailureClass`
- Add examples for unauthorized, forbidden, unknown persisted operation, and
  validation failure scenarios.

Add timing thresholds:

- Fixture-level `maxLatencyMs`.
- Suite-level default threshold.
- Markdown and JSON reports include duration per fixture and summary min/max or
  p50/p95 when useful.

Add retry policy:

- Configurable `retry` with max attempts, backoff, and retryable failure
  classes.
- Retry only network and timeout failures by default.
- Do not retry assertion, GraphQL error, or coverage failures by default.
- Report attempts clearly in evidence output.

Verification:

- `go test ./...`
- Tests for retry classification and non-retryable failures.
- Tests for latency threshold failure.
- Snapshot update and redaction tests.

## Affected Files And Docs

Expected package files:

- `cmd/graphgate/main.go`
- `cmd/graphgate/main_test.go`
- `internal/config/config.go`
- `internal/core/test_runner.go`
- `internal/core/test_runner_test.go`
- `internal/report/test.go`
- `docs/CONFIG_REFERENCE.md`
- `docs/CONTRACT_TESTS.md`
- `README.md`

Optional files if the implementation grows:

- `internal/core/assertions.go`
- `internal/core/evidence.go`
- `internal/core/readiness.go`
- `internal/core/selection.go`

## Risks

- Adding too many assertion features at once could make fixtures harder to
  understand. Keep each operator simple and test-driven.
- Evidence output could accidentally include sensitive values. Redaction must be
  part of the first evidence implementation, not a follow-up.
- Retry support can hide real failures. Keep default retries narrow and visible.
- JSONPath support can pull in a large dependency. Prefer a small internal path
  evaluator unless a dependency clearly improves maintainability.

## Rollback Or Recovery

- Each phase should be independently revertible.
- Keep new config keys optional so existing users can downgrade by removing new
  keys from their config.
- If evidence output or assertion expansion becomes too broad, leave the CLI
  surface in place and reduce implementation to documented, tested primitives.

## Verification Gates

Before each commit:

```sh
cd tools/graphgate
go test ./...
go run ./cmd/graphgate validate --config ../graphgate-emsi/graphgate.yaml
go run ./cmd/graphgate manifest --config ../graphgate-emsi/graphgate.yaml --check
go run ./cmd/graphgate report --config ../graphgate-emsi/graphgate.yaml --format markdown
```

When a local EMSI API is available, also run:

```sh
cd tools/graphgate
EMSI_GRAPHGATE_EVENT_DATE=2026-05-31 go run ./cmd/graphgate smoke --config ../graphgate-emsi/graphgate.yaml --env local --timeout 10s
```

## Execution Prompt

Use `$google-eng-practices` and work from `/Users/marlonjd/Developer/mobile/emsi_swift`. Implement the next GraphGate package phase from `tools/graphgate/docs/plans/2026-05-31-e2e-evidence-runner-plan.md` without adding EMSI-specific behavior to the GraphGate package. Keep changes small and backwards-compatible. Update GraphGate docs and tests with the implementation. Verify with `cd tools/graphgate && go test ./...`, then run GraphGate validate, manifest `--check`, and report against `../graphgate-emsi/graphgate.yaml`. If local API dependencies are available, run GraphGate smoke against the local Go endpoint; otherwise report the endpoint availability gap. Commit only relevant files with author `marlonjd <burak.karahan@mail.ru>` using a Conventional Commit message and push immediately after a successful commit.
