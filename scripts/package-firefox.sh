#!/usr/bin/env bash
# Package devlog browser extension for Firefox Add-ons (AMO)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EXTENSION_DIR="$PROJECT_ROOT/browser-extension"
FIREFOX_DIR="$EXTENSION_DIR/firefox"
DIST_DIR="$PROJECT_ROOT/dist"

echo "📦 Packaging devlog extension for Firefox Add-ons..."

# Create dist directory
mkdir -p "$DIST_DIR"

# Get version from VERSION file
VERSION=$(cat "$EXTENSION_DIR/VERSION")
echo "Version: $VERSION"

# Update manifest.json with version from VERSION file
sed -i '' "s/\"version\": \"[^\"]*\"/\"version\": \"$VERSION\"/" "$FIREFOX_DIR/manifest.json"

# Create temporary build directory
BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT

echo "Building extension in $BUILD_DIR..."

# Copy Firefox-specific manifest
cp "$FIREFOX_DIR/manifest.json" "$BUILD_DIR/"

# Copy Firefox-specific files
cp "$FIREFOX_DIR/background.js" "$BUILD_DIR/"
cp "$FIREFOX_DIR/content_script.js" "$BUILD_DIR/"
cp "$EXTENSION_DIR/page_inject.js" "$BUILD_DIR/"
cp "$FIREFOX_DIR/popup.html" "$BUILD_DIR/"
cp "$FIREFOX_DIR/popup.js" "$BUILD_DIR/"

# Copy icons
mkdir -p "$BUILD_DIR/icons"
cp "$EXTENSION_DIR/icons/"*.png "$BUILD_DIR/icons/"

# Create zip file (Firefox uses .xpi but .zip works too)
OUTPUT_FILE="$DIST_DIR/devlog-firefox-v${VERSION}.zip"
cd "$BUILD_DIR"
zip -r "$OUTPUT_FILE" ./*

echo "✅ Firefox extension packaged: $OUTPUT_FILE"
echo ""
echo "Next steps:"
echo "1. Go to https://addons.mozilla.org/developers/"
echo "2. Submit a New Add-on"
echo "3. Upload $OUTPUT_FILE"
echo "4. Mozilla will review the source code"
echo "5. Fill in listing details"
echo "6. Submit for review"
echo ""
echo "Note: Firefox requires source code review. Be prepared to provide:"
echo "- Link to GitHub repository"
echo "- Build instructions"
echo "- Any dependencies used"
