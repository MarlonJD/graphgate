# CI Usage

GraphGate is designed to fail fast in CI and print readable markdown without opening the UI.

## Validate Operations

```yaml
- name: Validate GraphQL operations
  run: graphgate validate --config graphgate.yaml
```

## Check Persisted Manifest

```yaml
- name: Check persisted manifest
  run: graphgate manifest --config graphgate.yaml --check
```

## Check Schema Diff

```yaml
- name: Check schema diff
  run: graphgate diff --config graphgate.yaml --base schema.base.graphql --format markdown
```

`graphgate diff` exits non-zero when a breaking schema change impacts committed operations.

## PR Comment Report

```yaml
- name: Write GraphGate report
  run: graphgate diff --config graphgate.yaml --base schema.base.graphql --output graphgate/reports/diff.md
```

The generated markdown is suitable for pull request comments or job summaries:

```yaml
- name: Publish job summary
  run: cat graphgate/reports/diff.md >> "$GITHUB_STEP_SUMMARY"
```

## GitHub Action

```yaml
- uses: MarlonJD/graphgate@v0.1.0
  with:
    command: validate
    config: graphgate.yaml
```

Diff example:

```yaml
- uses: MarlonJD/graphgate@v0.1.0
  with:
    command: diff
    config: graphgate.yaml
    base-schema: schema.base.graphql
```
