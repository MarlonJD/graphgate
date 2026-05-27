# REST Parity Mode

REST parity mode is the documented migration workflow for teams replacing REST
responses with GraphQL operations. It is not a separate CLI command yet; the
current workflow uses fixtures, saved responses, field mappings, and generated
reports.

## Inputs

- REST fixture response: the current REST response body.
- GraphQL operation response: the GraphQL response for the replacement
  operation.
- Field mapping file: explicit JSON paths that define equivalence.
- Diff report: the comparison output saved for review.

## Mapping File

Each mapping links one REST JSON path to one GraphQL JSON path:

```json
{
  "name": "city list parity",
  "mappings": [
    {
      "rest": "items[].id",
      "graphql": "data.cities[].id",
      "required": true
    },
    {
      "rest": "items[].name",
      "graphql": "data.cities[].name",
      "required": true
    }
  ]
}
```

## Recommended Workflow

1. Capture a REST fixture response from the existing endpoint.
2. Add or update the equivalent GraphQL operation.
3. Capture the GraphQL response from the local or staging GraphQL endpoint.
4. Write an explicit field mapping file.
5. Compare mapped values and save a diff report.
6. Add the GraphQL operation to GraphGate validation and manifest checks.

## EMSI Example

Example files are included under:

```text
examples/emsi/rest-parity/
```

These files document the target shape for a future automated parity diff runner:

- `rest-response.example.json`
- `graphql-response.example.json`
- `field-mapping.example.json`
- `diff-report.example.json`
