#!/bin/bash
set -e
VERSION="${1}"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
esac
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
esac

BINARY="terralink-${OS}-${ARCH}"
URL="https://github.com/segator/terralink/releases/download/v${VERSION}/${BINARY}"
echo "Downloading $URL"
curl -L "$URL" -o terralink

echo "Installing into PATH"
chmod +x terralink
echo "$(pwd)/terralink" >> $GITHUB_PATH

