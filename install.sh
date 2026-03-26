#!/bin/sh
set -e

REPO="thesatellite-ai/fetchr"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)      echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
  *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release tag
TAG=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
  echo "Failed to fetch latest release tag"
  exit 1
fi

VERSION="${TAG#v}"
FILENAME="fetchr_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${TAG}/${FILENAME}"

echo "Downloading fetchr ${VERSION} for ${OS}/${ARCH}..."
TMPDIR=$(mktemp -d)
curl -sL "$URL" -o "${TMPDIR}/${FILENAME}"

echo "Extracting..."
tar -xzf "${TMPDIR}/${FILENAME}" -C "${TMPDIR}"

echo "Installing to ${INSTALL_DIR}..."
sudo mv "${TMPDIR}/fetchr" "${INSTALL_DIR}/fetchr"
sudo chmod +x "${INSTALL_DIR}/fetchr"

rm -rf "$TMPDIR"

echo "fetchr ${VERSION} installed successfully!"
echo "Run 'fetchr --help' to get started."
