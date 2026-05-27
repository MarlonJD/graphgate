# Distribution

GraphGate distribution targets:

- GitHub Releases with checksums and build provenance attestation.
- Homebrew formula in `packaging/homebrew/graphgate.rb`.
- Linux/macOS install script at `scripts/install.sh`.
- Docker image at `ghcr.io/marlonjd/graphgate`.
- GitHub Action via `action.yml`.
- Downloadable Windows `.exe` inside the Windows release archive.

## macOS

```sh
brew install marlonjd/tap/graphgate
graphgate validate
```

Build from source when Go is already available:

```sh
go install github.com/MarlonJD/graphgate/cmd/graphgate@latest
```

The Homebrew formula includes release checksums and is published in
`MarlonJD/homebrew-tap`.

## Linux

Recommended for CI runners and developer machines:

```sh
curl -fsSL https://raw.githubusercontent.com/MarlonJD/graphgate/main/scripts/install.sh | sh
```

Pinned version and user-local install:

```sh
mkdir -p "$HOME/.local/bin"
curl -fsSL https://raw.githubusercontent.com/MarlonJD/graphgate/main/scripts/install.sh \
  | GRAPHGATE_VERSION=v0.1.1 INSTALL_DIR="$HOME/.local/bin" sh
```

Direct release archive:

```sh
curl -L -o graphgate.tar.gz https://github.com/MarlonJD/graphgate/releases/download/v0.1.1/graphgate_linux_amd64.tar.gz
tar -xzf graphgate.tar.gz
install graphgate /usr/local/bin/graphgate
```

Build from source when Go is already available:

```sh
go install github.com/MarlonJD/graphgate/cmd/graphgate@v0.1.1
```

## Windows

Download `graphgate_windows_amd64.tar.gz` from GitHub Releases and place `graphgate.exe` on your `PATH`.

## Docker

```sh
docker run --rm -v "$PWD:/work" ghcr.io/marlonjd/graphgate:v0.1.1 validate
docker run --rm -v "$PWD:/work" ghcr.io/marlonjd/graphgate:v0.1.1 manifest --check
```

## GitHub Action

```yaml
- uses: MarlonJD/graphgate@v0.1.1
  with:
    command: validate
    version: v0.1.1
```
