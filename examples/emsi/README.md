# EMSI Example

This example dogfoods GraphGate against the EMSI gqlgen schema without vendoring
GraphGate into the EMSI repository.

Expected checkout layout:

```text
emsi_swift/
  emsi_go_api/internal/graph/schema.graphqls
  tools/graphgate/
```

From `tools/graphgate`:

```sh
go run ./cmd/graphgate validate --config examples/emsi/graphgate.yaml
go run ./cmd/graphgate manifest --config examples/emsi/graphgate.yaml --check
go run ./cmd/graphgate report --config examples/emsi/graphgate.yaml --format markdown
```

The sample operations are intentionally small. They establish a real validation
surface for the current schema while EMSI operation extraction is still pending.
