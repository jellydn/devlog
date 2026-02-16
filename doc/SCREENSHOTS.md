# Screenshot Guide for Store Submission

This guide helps you create the required screenshots for Chrome Web Store and Firefox Add-ons submissions.

## Required Screenshots

Both stores require screenshots showing the extension in action. Here's what to capture:

### Chrome Web Store Requirements

- **Minimum**: 1 screenshot
- **Recommended**: 3-5 screenshots
- **Dimensions**: 
  - 1280x800 pixels (16:10 ratio) - Recommended
  - 640x400 pixels (16:10 ratio) - Alternative
- **Format**: PNG or JPEG
- **Max file size**: 8 MB per image

### Firefox Add-ons Requirements

- **Minimum**: 1 screenshot
- **Recommended**: 3-5 screenshots
- **Dimensions**: Any, but 1280x800 recommended for consistency
- **Format**: PNG or JPEG
- **Max file size**: 10 MB per image

## Screenshot Ideas

### 1. Extension Popup (Essential)

**What to show**: The extension popup displaying status and configuration

**Steps**:
1. Install the extension (Chrome or Firefox)
2. Register the native host: `devlog register`
3. Start devlog: `devlog up` with a sample devlog.yml
4. Click the extension icon in the toolbar
5. Take a screenshot of the popup showing:
   - "Browser logging enabled" status (green)
   - List of monitored URLs
   - devlog branding

**Dimensions**: Can be smaller, but upscale to 1280x800 for store submission

### 2. Console Logs Being Captured (Essential)

**What to show**: Browser console with logs being generated

**Steps**:
1. Open a webpage matching your URL pattern (e.g., http://localhost:3000)
2. Open browser DevTools (F12)
3. Navigate to Console tab
4. Generate some test logs:
   ```javascript
   console.log('Hello from devlog');
   console.warn('This is a warning');
   console.error('This is an error');
   ```
5. Ensure the extension icon is visible in the toolbar
6. Take screenshot showing:
   - Browser window with DevTools open
   - Console messages visible
   - Extension icon highlighted/circled
   - Clean, professional appearance

**Dimensions**: 1280x800 (crop browser window appropriately)

### 3. Configuration File (Recommended)

**What to show**: The devlog.yml configuration file

**Steps**:
1. Open devlog.yml in a text editor with syntax highlighting
2. Show a typical configuration with browser settings:
   ```yaml
   version: 1
   project: my-app
   logs_dir: ./logs
   
   browser:
     native_host: true
     file: browser/console.log
     levels: [error, warn, info, log]
     urls:
       - "http://localhost:*/*"
   ```
3. Take screenshot showing:
   - YAML configuration
   - Browser section highlighted
   - Clean, readable text

**Dimensions**: 1280x800

### 4. Log Files Output (Recommended)

**What to show**: Captured logs in the filesystem

**Steps**:
1. Run a devlog session that captures browser logs
2. Open the logs directory in a terminal or file manager
3. Show the log file contents with `cat` or `less`:
   ```bash
   cat logs/2026-02-14_10-30-45/browser/console.log
   ```
4. Display several log entries showing:
   - Timestamps
   - Log levels ([ERROR], [WARN], [LOG])
   - Actual log messages
5. Take screenshot showing:
   - Terminal with log output
   - File path visible
   - Multiple log entries
   - Professional appearance

**Dimensions**: 1280x800

### 5. Extension in Action - Full Workflow (Optional)

**What to show**: Split screen showing browser + logs simultaneously

**Steps**:
1. Split screen with browser on left, terminal on right
2. Browser shows your app with DevTools console open
3. Terminal shows `tail -f logs/.../browser/console.log`
4. Demonstrate real-time log capture
5. Take screenshot showing:
   - Live application running
   - Logs appearing in terminal in real-time
   - Both browser and logs visible
   - Extension icon in toolbar

**Dimensions**: 1280x800

## Screenshot Creation Workflow

### macOS

```bash
# Full screen
Cmd + Shift + 3

# Select area
Cmd + Shift + 4

# Window capture
Cmd + Shift + 4, then Space

# Screenshots saved to Desktop
```

### Linux

```bash
# Gnome Screenshot
gnome-screenshot -a  # Area selection
gnome-screenshot -w  # Window

# Or use Flameshot
flameshot gui
```

### Windows

```bash
# Snipping Tool
Win + Shift + S

# Or use Windows Snip & Sketch
```

## Post-Processing

### Resize to Required Dimensions

Using ImageMagick:
```bash
# Resize to 1280x800 (maintain aspect ratio)
convert screenshot.png -resize 1280x800 -background white -gravity center -extent 1280x800 screenshot-1280x800.png

# Or for exact dimensions (may distort)
convert screenshot.png -resize 1280x800! screenshot-1280x800.png
```

Using online tools:
- [Pixlr](https://pixlr.com/)
- [Photopea](https://www.photopea.com/)
- [Canva](https://www.canva.com/)

### Optimize File Size

Using ImageMagick:
```bash
# Compress PNG
convert screenshot.png -quality 85 -strip screenshot-optimized.png

# Convert to JPEG (smaller file size)
convert screenshot.png -quality 85 screenshot.jpg
```

Using online tools:
- [TinyPNG](https://tinypng.com/)
- [Squoosh](https://squoosh.app/)

## Screenshot Naming Convention

Use descriptive filenames:
```
devlog-extension-popup.png
devlog-console-capture.png
devlog-configuration.png
devlog-log-output.png
devlog-full-workflow.png
```

## Tips for Great Screenshots

1. **Clean Environment**
   - Close unnecessary browser tabs
   - Hide personal bookmarks/extensions
   - Use default browser theme or a clean theme

2. **Good Content**
   - Use realistic example data
   - Show meaningful log messages
   - Use proper syntax highlighting

3. **Highlight Key Features**
   - Circle or highlight the extension icon
   - Add arrows pointing to important UI elements (optional)
   - Use annotations sparingly

4. **Professional Appearance**
   - Use high contrast and readable text
   - Ensure proper lighting (if showing physical screen)
   - No distracting background elements

5. **Consistency**
   - Use the same browser/theme across screenshots
   - Maintain similar style and layout
   - Keep dimensions consistent

## Example Screenshot Annotations

For Chrome Web Store, you can add brief captions when uploading screenshots:

1. **Popup**: "Extension popup showing capture status and monitored URLs"
2. **Console**: "Real-time console log capture from localhost development"
3. **Config**: "Simple YAML configuration for browser log capture"
4. **Output**: "Logs written to local files with timestamps and levels"
5. **Workflow**: "Live log capture during local development"

## Promotional Images (Optional)

### Chrome Web Store Small Tile (440x280)

Create a promotional tile showing:
- devlog logo/icon
- Tagline: "Zero-Code Browser Log Capture"
- Key feature: "Local Development Logging"

### Chrome Web Store Marquee (1400x560)

Create a banner showing:
- devlog branding
- Screenshot of extension in action
- Key benefits listed
- "Available for Chrome, Brave, Edge"

You can create these using:
- Canva (free templates available)
- Figma
- Adobe Photoshop/Illustrator
- GIMP (free)

## Checklist Before Submission

- [ ] All screenshots are 1280x800 or 640x400 pixels
- [ ] Screenshots are PNG or JPEG format
- [ ] File sizes are under 8 MB (Chrome) / 10 MB (Firefox)
- [ ] At least 1 screenshot shows the extension popup
- [ ] Screenshots show real usage, not mockups
- [ ] No personal information visible in screenshots
- [ ] Screenshots have good lighting and contrast
- [ ] Text in screenshots is readable
- [ ] Extension icon is visible in toolbar screenshots

## Storage

Store your screenshots in a screenshots/ directory (not committed to git):

```
devlog/
├── screenshots/           # Not in git
│   ├── chrome/
│   │   ├── 1-popup.png
│   │   ├── 2-console.png
│   │   ├── 3-config.png
│   │   └── 4-output.png
│   ├── firefox/
│   │   └── (same as chrome)
│   └── promotional/
│       ├── small-tile.png
│       └── marquee.png
└── ...
```

## Need Help?

If you need help creating screenshots:
1. Open an issue on GitHub
2. Share your setup and what you're trying to capture
3. Community members can provide examples or guidance

---

Good luck with your store submission! 🚀
