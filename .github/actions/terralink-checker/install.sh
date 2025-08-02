#!/bin/bash
set -e
VERSION="${1:-0.2.1}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  msys*|cygwin*|mingw*|windows*) OS="windows" ;;
  *) echo "Unsupported OS: $OS" && exit 1 ;;
esac
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac
EXT=""
if [[ "$OS" == "windows" ]]; then
  EXT=".exe"
fi
BINARY="terralink-${VERSION}-${OS}-${ARCH}${EXT}"
URL="https://github.com/segator/terralink/releases/download/v${VERSION}/${BINARY}"
echo "Downloading $URL"
curl -L "$URL" -o terralink${EXT}
chmod +x terralink${EXT}

