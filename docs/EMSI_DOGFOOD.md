# EMSI Dogfood

GraphGate is not vendored into EMSI. For local dogfood work, clone GraphGate as
an ignored external repository:

```sh
cd /path/to/emsi_swift
git clone https://github.com/MarlonJD/graphgate tools/graphgate
```

The EMSI repository should ignore:

```gitignore
tools/graphgate/
```

## Example Config

The example config lives at:

```text
examples/emsi/graphgate.yaml
```

It is designed to work when GraphGate is cloned at `tools/graphgate/` inside the
EMSI checkout. From the GraphGate directory:

```sh
go run ./cmd/graphgate validate --config examples/emsi/graphgate.yaml
go run ./cmd/graphgate manifest --config examples/emsi/graphgate.yaml --check
go run ./cmd/graphgate report --config examples/emsi/graphgate.yaml --format markdown
```

The config validates:

```text
emsi_go_api/internal/graph/schema.graphqls
```

with sample operations for:

- `cities`
- `activeEmojis`
- `viewer`

EMSI does not currently store extracted `.graphql` operation files in the Swift
or Go projects. The sample operation files in this example are the first
contract fixtures for dogfooding GraphGate against the current gqlgen schema.

## Runtime Tests

`graphgate test --env local --config examples/emsi/graphgate.yaml` expects an
EMSI GraphQL endpoint at `http://localhost:8080/graphql`. Secrets must come from
environment variables referenced by the config and are not written to reports.

## REST Migration

The REST-to-GraphQL migration path is documented in
[REST_PARITY_MODE.md](REST_PARITY_MODE.md). Example mapping and response files
live under `examples/emsi/rest-parity/`.
