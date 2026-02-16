#!/usr/bin/env bash
# Validate screenshot dimensions for Chrome Web Store and Firefox Add-ons

set -e

echo "📸 Screenshot Validator for Store Submission"
echo "============================================"
echo ""

# Check if ImageMagick is installed
if ! command -v identify &>/dev/null; then
	echo "⚠️  ImageMagick not found. Install it to validate screenshots:"
	echo "   macOS: brew install imagemagick"
	echo "   Ubuntu: sudo apt-get install imagemagick"
	echo "   Or use an online tool like https://www.photopea.com/"
	echo ""
fi

SCREENSHOT_DIR="${1:-screenshots}"

if [ ! -d "$SCREENSHOT_DIR" ]; then
	echo "❌ Screenshot directory not found: $SCREENSHOT_DIR"
	echo ""
	echo "Create the directory and add your screenshots:"
	echo "   mkdir -p screenshots/chrome"
	echo "   mkdir -p screenshots/firefox"
	echo ""
	echo "Place your screenshots there and run:"
	echo "   $0 $SCREENSHOT_DIR"
	exit 1
fi

echo "Checking screenshots in: $SCREENSHOT_DIR"
echo ""

# Valid dimensions
VALID_WIDTHS=(1280 640)
VALID_HEIGHTS=(800 400)

valid_count=0
error_count=0

# Function to check if dimensions are valid
check_dimensions() {
	local width=$1
	local height=$2
	local file=$3

	local valid=false

	# Check 1280x800
	if [ "$width" -eq 1280 ] && [ "$height" -eq 800 ]; then
		valid=true
	fi

	# Check 640x400
	if [ "$width" -eq 640 ] && [ "$height" -eq 400 ]; then
		valid=true
	fi

	if [ "$valid" = true ]; then
		echo "   ✅ $file (${width}x${height})"
		((valid_count++))
	else
		echo "   ❌ $file (${width}x${height}) - Invalid dimensions!"
		echo "      Required: 1280x800 or 640x400"
		((error_count++))
	fi
}

# Find all image files
find "$SCREENSHOT_DIR" -type f \( -name "*.png" -o -name "*.jpg" -o -name "*.jpeg" \) | while read -r file; do
	if command -v identify &>/dev/null; then
		# Use ImageMagick to get dimensions
		dimensions=$(identify -format "%w %h" "$file" 2>/dev/null || echo "0 0")
		width=$(echo "$dimensions" | cut -d' ' -f1)
		height=$(echo "$dimensions" | cut -d' ' -f2)
		check_dimensions "$width" "$height" "$file"
	else
		echo "   ⚠️  $file (cannot validate - ImageMagick not installed)"
	fi
done

echo ""
echo "============================================"

if [ $error_count -eq 0 ]; then
	echo "✅ All screenshots have valid dimensions!"
else
	echo "❌ Found $error_count screenshots with invalid dimensions"
	echo ""
	echo "To fix, resize using ImageMagick:"
	echo "   convert screenshot.png -resize 1280x800 -background white -gravity center -extent 1280x800 screenshot-fixed.png"
	echo ""
	echo "Or use an online editor like https://www.photopea.com/"
fi

echo ""
echo "Requirements:"
echo "  • Chrome Web Store: 1280x800 or 640x400, PNG/JPEG, max 8MB"
echo "  • Firefox Add-ons: Any size (1280x800 recommended), PNG/JPEG, max 10MB"
echo "  • At least 1 screenshot required, up to 5 maximum"
echo ""

# Check file count
png_count=$(find "$SCREENSHOT_DIR" -type f -name "*.png" 2>/dev/null | wc -l)
jpg_count=$(find "$SCREENSHOT_DIR" -type f \( -name "*.jpg" -o -name "*.jpeg" \) 2>/dev/null | wc -l)
total=$((png_count + jpg_count))

if [ "$total" -eq 0 ]; then
	echo "⚠️  No screenshots found!"
	echo ""
	echo "You need at least 1 screenshot for store submission."
	echo "Follow the guide in doc/SCREENSHOTS.md to create screenshots."
elif [ "$total" -lt 3 ]; then
	echo "ℹ️  Found $total screenshot(s). Consider adding more (up to 5 recommended)."
else
	echo "✅ Found $total screenshot(s). Good!"
fi

echo ""
echo "Next steps:"
echo "  1. Take screenshots following doc/SCREENSHOTS.md"
echo "  2. Place them in $SCREENSHOT_DIR/"
echo "  3. Run this script again to validate"
echo "  4. Upload to Chrome Web Store / Firefox Add-ons"
