#!/usr/bin/env sh
set -eu

repo="MarlonJD/graphgate"
version="${GRAPHGATE_VERSION:-latest}"
install_dir="${INSTALL_DIR:-/usr/local/bin}"

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "graphgate install: missing required command: $1" >&2
    exit 1
  fi
}

need curl
need tar

if [ "$version" = "latest" ]; then
  latest_url="$(curl -fsSIL -o /dev/null -w '%{url_effective}' "https://github.com/${repo}/releases/latest")"
  version="${latest_url##*/}"
fi

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  *)
    echo "graphgate install: unsupported OS: $(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "graphgate install: unsupported architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

asset="graphgate_${os}_${arch}.tar.gz"
base_url="https://github.com/${repo}/releases/download/${version}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

curl -fsSL -o "${tmp_dir}/${asset}" "${base_url}/${asset}"
curl -fsSL -o "${tmp_dir}/checksums.txt" "${base_url}/checksums.txt"

checksum_line="$(grep "  ${asset}$" "${tmp_dir}/checksums.txt" || true)"
if [ -z "$checksum_line" ]; then
  echo "graphgate install: checksum not found for ${asset}" >&2
  exit 1
fi

printf '%s\n' "$checksum_line" > "${tmp_dir}/checksums.one"
if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmp_dir" && sha256sum -c checksums.one)
elif command -v shasum >/dev/null 2>&1; then
  (cd "$tmp_dir" && shasum -a 256 -c checksums.one)
else
  echo "graphgate install: missing sha256sum or shasum for checksum verification" >&2
  exit 1
fi

tar -xzf "${tmp_dir}/${asset}" -C "$tmp_dir"
mkdir -p "$install_dir"
cp "${tmp_dir}/graphgate" "${install_dir}/graphgate"
chmod 755 "${install_dir}/graphgate"

echo "graphgate ${version} installed to ${install_dir}/graphgate"
