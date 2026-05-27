# CI Usage

GraphGate is designed to fail fast in CI and print readable markdown without opening the UI.

## Install Paths

Use the GitHub Action on GitHub-hosted runners. It downloads a pinned release
binary and verifies the checksum.

```yaml
- uses: actions/checkout@v6
- uses: MarlonJD/graphgate@v0.1.1
  with:
    command: validate
    config: graphgate.yaml
    version: v0.1.1
```

Use the install script on generic Linux CI runners:

```sh
mkdir -p "$PWD/.bin"
curl -fsSL https://raw.githubusercontent.com/MarlonJD/graphgate/main/scripts/install.sh \
  | GRAPHGATE_VERSION=v0.1.1 INSTALL_DIR="$PWD/.bin" sh
export PATH="$PWD/.bin:$PATH"
graphgate validate --config graphgate.yaml
```

Use Docker when the runner already supports containers:

```sh
docker run --rm -v "$PWD:/work" ghcr.io/marlonjd/graphgate:v0.1.1 validate --config graphgate.yaml
```

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
- uses: MarlonJD/graphgate@v0.1.1
  with:
    command: validate
    config: graphgate.yaml
    version: v0.1.1
```

Diff example:

```yaml
- uses: MarlonJD/graphgate@v0.1.1
  with:
    command: diff
    config: graphgate.yaml
    base-schema: schema.base.graphql
    version: v0.1.1
```

Manifest check example:

```yaml
- uses: MarlonJD/graphgate@v0.1.1
  with:
    command: manifest
    config: graphgate.yaml
    args: --check
    version: v0.1.1
```

## GitLab CI

```yaml
graphgate:
  image: alpine:3.20
  before_script:
    - apk add --no-cache curl tar
    - mkdir -p "$CI_PROJECT_DIR/.bin"
    - curl -fsSL https://raw.githubusercontent.com/MarlonJD/graphgate/main/scripts/install.sh | GRAPHGATE_VERSION=v0.1.1 INSTALL_DIR="$CI_PROJECT_DIR/.bin" sh
    - export PATH="$CI_PROJECT_DIR/.bin:$PATH"
  script:
    - graphgate validate --config graphgate.yaml
    - graphgate manifest --config graphgate.yaml --check
```
