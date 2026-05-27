# GraphGate

Catch GraphQL contract breaks before they ship.

GraphGate is a CLI-first GraphQL contract gate with a local web UI planned for a later milestone. It validates schemas, operation files, persisted operation manifests, and release compatibility in CI/CD before broken GraphQL contracts reach production.

## What It Is

- A local-first CLI for GraphQL schema and operation validation.
- A deterministic persisted-operation manifest generator.
- A CI/CD gate with stable exit codes and machine-readable reports.
- A future local browser UI started with `graphgate ui`.

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
graphgate report --format markdown
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
graphgate report --format markdown
graphgate report --format json
```

Planned later:

```sh
graphgate diff
graphgate test
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

See [docs/CONFIG_REFERENCE.md](docs/CONFIG_REFERENCE.md) for the full M1 config reference.

## License

MIT
