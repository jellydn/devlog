# devlog Browser Extension

Browser extension for capturing console logs and sending them to the devlog native host for persistent local logging.

## Features

- Captures `console.log`, `console.error`, `console.warn`, `console.info`, `console.debug`
- Captures uncaught errors and unhandled promise rejections
- URL pattern filtering (only logs from matching URLs)
- Native messaging integration with devlog-host
- Zero code changes required in your applications

## Directory Structure

```
browser-extension/
├── chrome/              # Chrome-specific manifest (Manifest V3)
│   └── manifest.json
├── firefox/             # Firefox-specific manifest (Manifest V2)
│   └── manifest.json
├── icons/               # Extension icons
│   ├── icon.svg        # Original SVG icon
│   ├── icon16.png      # 16x16 PNG
│   ├── icon32.png      # 32x32 PNG
│   ├── icon48.png      # 48x48 PNG
│   └── icon128.png     # 128x128 PNG
├── background.js        # Background service worker / script
├── content_script.js    # Injected into web pages
├── page_inject.js       # Injected into page context
├── popup.html          # Extension popup UI
└── popup.js            # Popup logic
```

## Installation

### Chrome / Brave (Development)

1. Open `chrome://extensions/` or `brave://extensions/`
2. Enable "Developer mode"
3. Click "Load unpacked"
4. Select the `browser-extension/chrome` directory

### Firefox (Development)

1. Open `about:debugging#/runtime/this-firefox`
2. Click "Load Temporary Add-on"
3. Select any file in `browser-extension/firefox` directory

### From Store (Coming Soon)

- **Chrome Web Store**: [Link will be added after publication]
- **Firefox Add-ons**: [Link will be added after publication]

## Configuration

The extension reads configuration from the devlog native host, which gets settings from your `devlog.yml`:

```yaml
browser:
  native_host: true
  file: browser/console.log
  levels: [error, warn, info, log]
  urls:
    - "http://localhost:*/*"
    - "https://localhost:*/*"
```

## How It Works

1. **Content Script** (`content_script.js`) is injected into matching web pages
2. **Page Inject** (`page_inject.js`) runs in the page context to capture console output
3. **Background Script** (`background.js`) receives logs via `window.postMessage`
4. **Native Messaging** sends logs to `devlog-host` binary
5. **devlog-host** writes logs to local files

## Development

### Building Icons

To regenerate PNG icons from the SVG source:

```bash
cd icons
rsvg-convert -w 16 -h 16 icon.svg -o icon16.png
rsvg-convert -w 32 -h 32 icon.svg -o icon32.png
rsvg-convert -w 48 -h 48 icon.svg -o icon48.png
rsvg-convert -w 128 -h 128 icon.svg -o icon128.png
```

### Packaging for Stores

Package for Chrome Web Store:
```bash
just package-chrome
# or
./scripts/package-chrome.sh
```

Package for Firefox Add-ons:
```bash
just package-firefox
# or
./scripts/package-firefox.sh
```

Package both:
```bash
just package-extensions
```

### Testing Changes

After making changes to shared files (background.js, content_script.js, etc.):

1. **Chrome**: Go to `chrome://extensions/` and click the reload icon
2. **Firefox**: Go to `about:debugging` and click "Reload"

### Debugging

**Chrome DevTools**:
- Background script: Click "service worker" link in `chrome://extensions/`
- Content script: Right-click page → Inspect → Console tab → Select content script from dropdown
- Popup: Right-click extension icon → Inspect popup

**Firefox DevTools**:
- Background script: `about:debugging` → This Firefox → Inspect
- Content script: Regular page inspector, content script shows in debugger
- Popup: Right-click extension icon → Inspect popup

## Browser Compatibility

### Chrome / Chromium / Brave / Edge

- **Manifest Version**: V3
- **Minimum Version**: Chrome 109+
- **Status**: ✅ Fully supported

### Firefox

- **Manifest Version**: V2
- **Minimum Version**: Firefox 109+
- **Status**: ✅ Fully supported
- **Note**: Manifest V3 support planned for future versions

## Permissions

The extension requires these permissions:

- **`nativeMessaging`**: Communication with devlog-host binary
- **`storage`**: Store configuration locally
- **`activeTab`**: Read console logs from active tabs
- **`<all_urls>`**: Inject content script on user-configured URLs only

See [PRIVACY.md](../PRIVACY.md) for details on data handling.

## Architecture

```
┌──────────────┐
│  Web Page    │
└──────┬───────┘
       │ console.log()
       ▼
┌──────────────┐
│ page_inject  │ (page context)
└──────┬───────┘
       │ postMessage
       ▼
┌──────────────┐
│content_script│ (isolated world)
└──────┬───────┘
       │ chrome.runtime.sendMessage
       ▼
┌──────────────┐
│ background   │ (service worker)
└──────┬───────┘
       │ chrome.runtime.sendNativeMessage
       ▼
┌──────────────┐
│ devlog-host  │ (native binary)
└──────┬───────┘
       │ write
       ▼
┌──────────────┐
│  Log Files   │
└──────────────┘
```

## Troubleshooting

### Extension not capturing logs

1. **Check native host registration**:
   ```bash
   devlog healthcheck
   ```

2. **Verify URL patterns** in devlog.yml match the page you're testing

3. **Check browser console** for extension errors

### Native messaging errors

- **Error: "Specified native messaging host not found"**
  - Run `devlog register` to register the native host
  - Verify `devlog-host` binary is in your PATH

- **Error: "Native host has exited"**
  - Check devlog-host logs
  - Ensure devlog CLI is running (`devlog up`)

### Popup shows "disabled"

- Start devlog session: `devlog up`
- Check that browser.native_host is true in devlog.yml

## Contributing

Contributions are welcome! Please:

1. Test changes in both Chrome and Firefox
2. Update both manifest files if needed
3. Follow existing code style
4. Test with `devlog up` and real applications

## License

MIT License - See [LICENSE](../LICENSE)

## Related Documentation

- [Store Submission Guide](../doc/STORE_SUBMISSION.md)
- [Privacy Policy](../PRIVACY.md)
- [Main README](../README.md)
