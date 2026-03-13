#!/bin/sh
set -eu

REPO="${REPO:-yanickxia/share-info}"
BIN_NAME="share-info"
OUT_FILE="${OUT_FILE:-env.snapshot.enc.b64}"

if ! command -v curl >/dev/null 2>&1; then
  echo "[error] curl not found" >&2
  exit 1
fi
if ! command -v tar >/dev/null 2>&1; then
  echo "[error] tar not found" >&2
  exit 1
fi

os_name="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os_name" in
  linux*)
    os="linux"
    ;;
  darwin*)
    os="darwin"
    ;;
  *)
    echo "[error] unsupported OS: $os_name" >&2
    exit 1
    ;;
esac

arch_name="$(uname -m | tr '[:upper:]' '[:lower:]')"
case "$arch_name" in
  x86_64|amd64)
    arch="amd64"
    ;;
  aarch64|arm64)
    arch="arm64"
    ;;
  *)
    echo "[error] unsupported arch: $arch_name" >&2
    exit 1
    ;;
esac

archive="${BIN_NAME}-${os}-${arch}.tar.gz"
url="https://github.com/${REPO}/releases/latest/download/${archive}"

tmpdir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

printf '[info] download: %s\n' "$url"
curl -fL "$url" -o "$tmpdir/$archive"

tar -xzf "$tmpdir/$archive" -C "$tmpdir"
bin_path="$tmpdir/${BIN_NAME}-${os}-${arch}"
chmod +x "$bin_path"

if [ "$#" -eq 0 ]; then
  if [ -z "${ENV_SNAPSHOT_PASSWORD:-}" ]; then
    cat >&2 <<'USAGE'
[error] ENV_SNAPSHOT_PASSWORD is empty.
Set password first, e.g.
  curl -fsSL https://raw.githubusercontent.com/yanickxia/share-info/main/scripts/run-latest.sh | ENV_SNAPSHOT_PASSWORD='your-pass' sh
USAGE
    exit 2
  fi

  printf '[info] run default: -mode encrypt -out %s\n' "$OUT_FILE"
  "$bin_path" -mode encrypt -out "$OUT_FILE"
else
  "$bin_path" "$@"
fi
