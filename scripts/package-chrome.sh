#!/usr/bin/env bash
# Package devlog browser extension for Chrome Web Store

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EXTENSION_DIR="$PROJECT_ROOT/browser-extension"
CHROME_DIR="$EXTENSION_DIR/chrome"
DIST_DIR="$PROJECT_ROOT/dist"

echo "📦 Packaging devlog extension for Chrome Web Store..."

# Create dist directory
mkdir -p "$DIST_DIR"

# Get version from manifest
VERSION=$(grep -o '"version": "[^"]*"' "$CHROME_DIR/manifest.json" | cut -d'"' -f4)
echo "Version: $VERSION"

# Create temporary build directory
BUILD_DIR=$(mktemp -d)
trap "rm -rf $BUILD_DIR" EXIT

echo "Building extension in $BUILD_DIR..."

# Copy Chrome-specific manifest
cp "$CHROME_DIR/manifest.json" "$BUILD_DIR/"

# Copy shared files
cp "$EXTENSION_DIR/background.js" "$BUILD_DIR/"
cp "$EXTENSION_DIR/content_script.js" "$BUILD_DIR/"
cp "$EXTENSION_DIR/page_inject.js" "$BUILD_DIR/"
cp "$EXTENSION_DIR/popup.html" "$BUILD_DIR/"
cp "$EXTENSION_DIR/popup.js" "$BUILD_DIR/"

# Copy icons
mkdir -p "$BUILD_DIR/icons"
cp "$EXTENSION_DIR/icons/"*.png "$BUILD_DIR/icons/"

# Create zip file
OUTPUT_FILE="$DIST_DIR/devlog-chrome-v${VERSION}.zip"
cd "$BUILD_DIR"
zip -r "$OUTPUT_FILE" ./*

echo "✅ Chrome extension packaged: $OUTPUT_FILE"
echo ""
echo "Next steps:"
echo "1. Go to https://chrome.google.com/webstore/devconsole/"
echo "2. Create a new item or update existing"
echo "3. Upload $OUTPUT_FILE"
echo "4. Fill in store listing details"
echo "5. Submit for review"
