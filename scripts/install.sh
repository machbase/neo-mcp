#!/bin/sh
set -eu

repository="machbase/neo-mcp"
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)

case "$os" in
  linux|darwin) ;;
  *) echo "Unsupported operating system: $os" >&2; exit 1 ;;
esac

case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  aarch64|arm64) arch=arm64 ;;
  *) echo "Unsupported architecture: $arch" >&2; exit 1 ;;
esac

archive="neo-mcp_${os}_${arch}.tar.gz"
release=${NEO_MCP_VERSION:-latest}
if [ "$release" = "latest" ]; then
  release_path="releases/latest/download"
else
  release_path="releases/download/${release}"
fi
url="https://github.com/${repository}/${release_path}/${archive}"
destination=${NEO_MCP_INSTALL_DIR:-"$HOME/.local/bin"}
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT HUP INT TERM

mkdir -p "$destination"
curl -fsSL --retry 3 "$url" -o "$temporary/$archive"
tar -xzf "$temporary/$archive" -C "$temporary"
install -m 0755 "$temporary/neo-mcp_${os}_${arch}/neo-mcp" "$destination/neo-mcp"
printf 'Installed neo-mcp to %s\n' "$destination/neo-mcp"
