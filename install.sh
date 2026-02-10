#!/bin/sh
set -e

REPO="jellydn/devlog"

# Default to ~/.local/bin (user-writable), override with INSTALL_DIR
if [ -z "$INSTALL_DIR" ]; then
  INSTALL_DIR="${HOME}/.local/bin"
  if [ -d /usr/local/bin ] && touch /usr/local/bin/.devlog_write_test 2>/dev/null; then
    rm -f /usr/local/bin/.devlog_write_test
    INSTALL_DIR="/usr/local/bin"
  fi
fi

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
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
fi

if [ -z "$VERSION" ]; then
  echo "Error: Could not determine latest version." >&2
  echo "No releases found at https://github.com/${REPO}/releases" >&2
  echo "" >&2
  echo "To install from source instead:" >&2
  echo "  go install github.com/${REPO}/cmd/devlog@latest" >&2
  echo "  go install github.com/${REPO}/cmd/devlog-host@latest" >&2
  exit 1
fi

FILENAME="devlog_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo "Installing devlog ${VERSION} (${OS}/${ARCH})..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${FILENAME}"
tar -xzf "${TMPDIR}/${FILENAME}" -C "$TMPDIR"

mkdir -p "${INSTALL_DIR}"
cp "${TMPDIR}/devlog" "${INSTALL_DIR}/devlog"
cp "${TMPDIR}/devlog-host" "${INSTALL_DIR}/devlog-host"
chmod 755 "${INSTALL_DIR}/devlog" "${INSTALL_DIR}/devlog-host"

echo "Installed devlog to ${INSTALL_DIR}/devlog"
echo "Installed devlog-host to ${INSTALL_DIR}/devlog-host"

# Check if INSTALL_DIR is in PATH
case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo ""
    echo "NOTE: ${INSTALL_DIR} is not in your PATH. Add it with:"
    echo "  export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

echo ""
echo "Run 'devlog help' to get started."
