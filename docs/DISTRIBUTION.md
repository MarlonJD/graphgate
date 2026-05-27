# Distribution

GraphGate distribution targets:

- GitHub Releases with checksums and build provenance attestation.
- Homebrew formula template in `packaging/homebrew/graphgate.rb`.
- Docker image at `ghcr.io/marlonjd/graphgate`.
- GitHub Action via `action.yml`.
- Downloadable Windows `.exe` inside the Windows release archive.

## macOS

```sh
brew install marlonjd/tap/graphgate
graphgate validate
```

Until the tap is published:

```sh
go install github.com/MarlonJD/graphgate/cmd/graphgate@latest
```

The Homebrew formula contains release checksum placeholders until the first
published release. Replace each `sha256` value from the generated checksum file
before publishing the tap.

## Linux

```sh
curl -L -o graphgate.tar.gz https://github.com/MarlonJD/graphgate/releases/download/v0.1.0/graphgate_linux_amd64.tar.gz
tar -xzf graphgate.tar.gz
install graphgate /usr/local/bin/graphgate
```

## Windows

Download `graphgate_windows_amd64.tar.gz` from GitHub Releases and place `graphgate.exe` on your `PATH`.

## Docker

```sh
docker run --rm -v "$PWD:/work" ghcr.io/marlonjd/graphgate validate
docker run --rm -v "$PWD:/work" ghcr.io/marlonjd/graphgate manifest --check
```

## GitHub Action

```yaml
- uses: MarlonJD/graphgate@v0.1.0
  with:
    command: validate
```
