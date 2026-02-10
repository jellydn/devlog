# devlog Browser Extension

Browser extension for capturing console logs and sending them to the devlog native host.

## Supported Browsers

- **Chrome**: Manifest V3
- **Firefox**: Manifest V2 (also compatible with V3)

## Installation

### Chrome

1. Open Chrome and navigate to `chrome://extensions/`
2. Enable "Developer mode" (toggle in top right)
3. Click "Load unpacked"
4. Select the `browser-extension` directory
5. The extension icon should appear in your toolbar

### Firefox

1. Open Firefox and navigate to `about:debugging`
2. Click "This Firefox" (or "This Nightly" for Developer Edition)
3. Click "Load Temporary Add-on"
4. Select the `manifest-firefox.json` file in the `browser-extension` directory
5. The extension icon should appear in your toolbar

## Files

- `manifest.json` - Chrome Manifest V3
- `manifest-firefox.json` - Firefox Manifest V2
- `background.js` - Background script managing native messaging connection
- `content_script.js` - Injected into web pages to capture console logs
- `popup.html` / `popup.js` - Extension popup UI
- `icons/icon.svg` - Extension icon

## Features

- Captures all `console.*` methods (log, info, warn, error, debug, trace)
- Captures uncaught JavaScript errors
- Captures unhandled promise rejections
- Filters logs by URL patterns from devlog.yml
- Filters logs by log levels
- Sends logs to native host via Native Messaging API
- Works in all frames (iframes included)

## Native Messaging

The extension communicates with the `devlog-host` binary via Chrome's Native Messaging API. The native host manifest must be registered with the browser before the extension can connect.

### Native Host Manifest

The native host manifest file (`com.devlog.host.json`) must be placed in the browser's native messaging directory:

**Chrome (macOS):**

```
~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.devlog.host.json
```

**Chrome (Linux):**

```
~/.config/google-chrome/NativeMessagingHosts/com.devlog.host.json
```

**Firefox (macOS):**

```
~/Library/Application Support/Mozilla/NativeMessagingHosts/com.devlog.host.json
```

**Firefox (Linux):**

```
~/.mozilla/native-messaging-hosts/com.devlog.host.json
```

### Manifest Format

```json
{
  "name": "com.devlog.host",
  "description": "devlog Native Messaging Host",
  "path": "/absolute/path/to/devlog-host",
  "type": "stdio",
  "allowed_origins": ["chrome-extension://<extension-id>/"]
}
```

For Firefox, replace `allowed_origins` with:

```json
"allowed_extensions": ["devlog@example.com"]
```

## Configuration

The extension receives configuration from the devlog CLI via the `UPDATE_CONFIG` message. Configuration includes:

- `enabled`: Whether logging is enabled
- `urls`: Array of URL patterns to monitor (e.g., `http://localhost:3000/*`)
- `levels`: Array of log levels to capture
- `file`: Path to the browser log file (used by native host)

## Development

### Testing

1. Load the extension in your browser (see Installation)
2. Open a webpage that matches your configured URLs
3. Open browser console and run some console commands
4. Check the browser.log file in your logs directory

### Debugging

- Extension logs can be viewed in the browser's extension debugger:
  - Chrome: `chrome://extensions/` → "Inspect views: service worker"
  - Firefox: `about:debugging` → "Inspect"
- Content script logs appear in the webpage's console (prefixed with "devlog:")

### Building

No build step required! The extension is pure JavaScript/CSS/HTML.

## Protocol

### Content Script → Background

Content scripts send log messages to the background script:

```javascript
{
  type: "LOG",
  level: "error",
  url: "http://localhost:3000/page",
  source: "app.js",
  line: "42",
  column: "15",
  message: "Uncaught Error: Something went wrong",
  timestamp: "2024-01-15T10:30:00.000Z"
}
```

### Background → Native Host

Background script forwards logs to native host (same format as above).

### Native Host → Background

Native host sends acknowledgments:

```javascript
{
  type: "ACK",
  success: true
}
```

## License

MIT - See LICENSE file in parent directory
