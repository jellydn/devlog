#!/bin/sh
set -e

REPO="jellydn/devlog"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get version
if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$VERSION" ]; then
  echo "Error: Could not determine latest version" >&2
  exit 1
fi

VERSION_NO_V="${VERSION#v}"
FILENAME="devlog_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo "Installing devlog ${VERSION} (${OS}/${ARCH})..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${FILENAME}"
tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"

install -m 755 "${TMPDIR}/devlog" "${INSTALL_DIR}/devlog"
install -m 755 "${TMPDIR}/devlog-host" "${INSTALL_DIR}/devlog-host"

echo "Installed devlog to ${INSTALL_DIR}/devlog"
echo "Installed devlog-host to ${INSTALL_DIR}/devlog-host"
echo ""
echo "Run 'devlog help' to get started."
