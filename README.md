# GraphGate

Catch GraphQL contract breaks before they ship.

GraphGate is a CLI-first GraphQL contract gate with a local browser UI. It validates schemas, operation files, persisted operation manifests, and release compatibility in CI/CD before broken GraphQL contracts reach production.

## Why You Need GraphGate

GraphQL gives clients a typed contract, but that contract is easy to break in
ordinary delivery work: a field is removed, a resolver changes behavior, an
operation is not updated, or a persisted-operation manifest is regenerated
locally but not committed. These failures often reach staging or production
because schema tools focus on exploration, not release gates.

GraphGate gives teams a repeatable contract check that can run locally and in
CI. It answers practical release questions before code ships:

- Do all committed operations still validate against the schema?
- Is the persisted-operation manifest deterministic and up to date?
- Did a schema change break an operation that a mobile or frontend client uses?
- Do fixture-based GraphQL calls still return the expected status, error codes,
  and response shape?
- Can a developer inspect the same validation, manifest, diff, and test state
  locally without using a hosted graph platform?

Use GraphGate when GraphQL is part of your release contract, especially for
mobile apps, frontend clients, persisted operations, schema migrations, or
REST-to-GraphQL migration work. See
[docs/GRAPHGATE_GUIDE.md](docs/GRAPHGATE_GUIDE.md) for the detailed guide.

## What It Is

- A local-first CLI for GraphQL schema and operation validation.
- A deterministic persisted-operation manifest generator.
- A CI/CD gate with stable exit codes and machine-readable reports.
- A local browser UI started with `graphgate ui`.

## What It Is Not

- Not a hosted GraphQL platform.
- Not a desktop app.
- Not a replacement for GraphiQL, Altair, or Apollo Explorer. Those tools help humans explore and run GraphQL. GraphGate is focused on automated contract checks, release safety, manifests, and CI output.

## Install From Source

```sh
go install github.com/MarlonJD/graphgate/cmd/graphgate@latest
```

During local development:

```sh
go run ./cmd/graphgate validate
```

## Quick Start

```sh
graphgate init
graphgate validate
graphgate manifest
graphgate manifest --check
graphgate diff --base schema.base.graphql
graphgate test --env local
graphgate report --format markdown
graphgate ui
```

`graphgate init` creates:

- `graphgate.yaml`
- `operations/`
- `graphgate/fixtures/`
- `graphgate/reports/`

## Commands

```sh
graphgate init
graphgate validate
graphgate manifest
graphgate manifest --check
graphgate diff --base schema.base.graphql
graphgate test --env local
graphgate report --format markdown
graphgate report --format json
graphgate ui
```

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

## Configuration

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

See [docs/CONFIG_REFERENCE.md](docs/CONFIG_REFERENCE.md) for the full config reference.

## CI/CD Gate

GraphGate can fail CI when operations no longer validate, when the persisted
manifest is stale, or when a schema diff impacts committed operations:

```sh
graphgate validate
graphgate manifest --check
graphgate diff --base schema.base.graphql
```

See [docs/CI.md](docs/CI.md) for GitHub Actions examples and PR-comment-ready
markdown reports.

## Contract Tests

```sh
graphgate test --env local
graphgate test --env staging --format json
graphgate test --env local --update
```

Fixtures support variables, expected HTTP status, expected GraphQL error codes,
JSON shape assertions, and snapshots. See
[docs/CONTRACT_TESTS.md](docs/CONTRACT_TESTS.md).

## Local Web UI

```sh
graphgate ui
```

The browser UI defaults to `http://localhost:4317`, reads the same config as the
CLI, and never requires cloud login. See [docs/UI.md](docs/UI.md).

## EMSI Dogfood

GraphGate can be cloned beside EMSI at `tools/graphgate/` without vendoring it
into the EMSI repository. The EMSI dogfood example validates
`emsi_go_api/internal/graph/schema.graphqls` with sample operations. See
[docs/EMSI_DOGFOOD.md](docs/EMSI_DOGFOOD.md).

## Distribution

Install targets include GitHub Releases, Homebrew, Docker, GitHub Actions, and a
downloadable Windows executable. See [docs/DISTRIBUTION.md](docs/DISTRIBUTION.md).

## License

Copyright (C) 2026 Burak Karahan.

GraphGate is licensed under the GNU Affero General Public License v3.0 or later
(`AGPL-3.0-or-later`). The project uses AGPL to keep GraphGate local-first and
open while making sure meaningful forks and service-based variants share their
source changes with the community.
