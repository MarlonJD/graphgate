# Product Plan

GraphGate is a CLI-first GraphQL contract gate with a local browser UI.

## Positioning

Catch GraphQL contract breaks before they ship.

GraphGate gives teams a repeatable local and CI workflow for schema, operation, manifest, and release compatibility checks. It is designed for projects that need reliable GraphQL automation without adopting a hosted graph platform.

## Target Users

- Backend engineers maintaining GraphQL schemas.
- Mobile and frontend engineers depending on persisted GraphQL operations.
- Platform engineers wiring GraphQL checks into CI/CD.
- Teams migrating between GraphQL implementations or validating release compatibility.

## Primary Use Cases

- Validate operation files against a schema in CI.
- Generate deterministic persisted-operation manifests.
- Fail pull requests when committed manifests are stale.
- Report schema changes that break committed operations.
- Run fixture-based GraphQL contract tests against local or staging endpoints.
- Inspect validation, manifest, diff, and test state in a local browser UI.

## Non-Goals

- Hosted cloud service.
- Desktop application.
- GraphQL IDE replacement.
- Query authoring experience competing with GraphiQL, Altair, or Apollo Explorer.
- Schema registry as a mandatory dependency.

## Differentiation

GraphiQL, Altair, and Apollo Explorer are excellent interactive exploration tools. GraphGate focuses on automated release safety:

- CI-first exit codes.
- Deterministic manifest generation and checking.
- Operation impact reports for schema changes.
- Local-first behavior with no required cloud login.
- Local browser UI for inspecting the same data produced by CLI runs.
