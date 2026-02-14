# Browser Extension Store Submission Guide

This guide walks you through publishing the devlog browser extension to Chrome Web Store and Firefox Add-ons.

## Prerequisites

Before submitting to either store, ensure you have:

- [x] Privacy policy (see [PRIVACY.md](../PRIVACY.md))
- [x] Extension icons in multiple sizes (16x16, 32x32, 48x48, 128x128)
- [x] Updated manifest files with store metadata
- [x] Packaging scripts for both stores
- [ ] Screenshots of the extension in action (see [Screenshots](#screenshots) below)
- [ ] Store listing copy (see [Store Listings](#store-listings) below)
- [ ] Developer accounts for both stores

## Chrome Web Store Submission

### 1. Create Developer Account

1. Go to [Chrome Web Store Developer Dashboard](https://chrome.google.com/webstore/devconsole/)
2. Sign in with your Google account
3. Pay the one-time $5 developer registration fee
4. Agree to the developer agreement

### 2. Package the Extension

```bash
./scripts/package-chrome.sh
```

This creates `dist/devlog-chrome-v1.0.0.zip`

### 3. Create Store Listing

1. Click "New Item" in the developer dashboard
2. Upload `dist/devlog-chrome-v1.0.0.zip`
3. Fill in the store listing (see [Chrome Store Listing](#chrome-store-listing) below)

### 4. Required Assets

Upload these assets in the developer dashboard:

#### Store Icon
- Size: 128x128 pixels
- File: `browser-extension/icons/icon128.png`

#### Screenshots
- At least 1 screenshot required (1280x800 or 640x400)
- See [Screenshots](#screenshots) section below

#### Promotional Images (Optional but Recommended)
- Small tile: 440x280 pixels
- Marquee: 1400x560 pixels

### 5. Privacy & Permissions

- **Privacy Policy URL**: `https://github.com/jellydn/devlog/blob/main/PRIVACY.md`
- **Justification for Permissions**:
  - `nativeMessaging`: Required to communicate with local devlog-host application
  - `storage`: Stores user configuration for URL patterns and log levels
  - `activeTab`: Reads console logs from active tab
  - `<all_urls>`: Required to inject log capture on user-configured URLs

### 6. Submit for Review

1. Review all fields
2. Click "Submit for Review"
3. Review typically takes 1-3 business days

## Firefox Add-ons Submission

### 1. Create Developer Account

1. Go to [Firefox Add-on Developer Hub](https://addons.mozilla.org/developers/)
2. Sign in with your Firefox account
3. Read and agree to the developer agreement (no fee required)

### 2. Package the Extension

```bash
./scripts/package-firefox.sh
```

This creates `dist/devlog-firefox-v1.0.0.zip`

### 3. Submit Add-on

1. Click "Submit a New Add-on"
2. Choose "On this site"
3. Upload `dist/devlog-firefox-v1.0.0.zip`

### 4. Source Code Review

Firefox requires source code review for all extensions. Be prepared to provide:

**Source Code Location**: Point to the GitHub repository:
```
https://github.com/jellydn/devlog
```

**Build Instructions**:
```
The extension does not require a build step. All source files are included in the package:
- background.js
- content_script.js
- page_inject.js
- popup.html
- popup.js
- manifest.json

To package manually:
cd browser-extension/firefox
zip -r devlog-firefox.zip manifest.json background.js content_script.js page_inject.js popup.html popup.js icons/
```

**Dependencies**: None (vanilla JavaScript only)

### 5. Fill in Listing Details

See [Firefox Store Listing](#firefox-store-listing) below

### 6. Privacy & Permissions

- **Privacy Policy URL**: `https://github.com/jellydn/devlog/blob/main/PRIVACY.md`
- **Does this add-on collect user data?**: No

### 7. Submit for Review

1. Review all fields
2. Submit for review
3. Review typically takes 1-5 business days (longer for first submission)

## Screenshots

Create screenshots showing the extension in action. Recommended screenshots:

### Screenshot 1: Extension Popup
- Show the extension popup with status and monitored URLs
- Dimensions: 1280x800 or 640x400

### Screenshot 2: Console Logs Being Captured
- Show browser console with logs
- Highlight the extension icon in toolbar
- Dimensions: 1280x800

### Screenshot 3: Log Files on Disk
- Show terminal or file manager with captured log files
- Demonstrate the timestamped directory structure
- Dimensions: 1280x800

### How to Take Screenshots

1. Load the extension in your browser
2. Open a test website that generates console logs
3. Click the extension icon to show the popup
4. Use browser or OS screenshot tools
5. Crop to appropriate dimensions

## Store Listings

### Chrome Store Listing

**Name**: devlog - Browser Log Capture

**Short Description** (132 characters max):
```
Zero-code local development log capture. Captures browser console logs to local files via native messaging.
```

**Detailed Description**:
```
devlog is a developer tool that captures browser console logs and writes them to local log files for debugging. Perfect for local development workflows where you need persistent logs.

✨ FEATURES

• Zero code changes - no SDK or instrumentation required
• Captures console.log, console.error, console.warn, console.info, console.debug
• Captures uncaught errors and unhandled promise rejections
• URL pattern filtering - only capture logs from specific URLs
• Timestamped log entries
• Works with tmux sessions for comprehensive dev logging
• Completely local - no data sent to external servers

🔧 HOW IT WORKS

1. Configure URL patterns in devlog.yml
2. Extension captures console logs from matching pages
3. Logs are sent via Native Messaging to devlog-host
4. Logs are written to local files on your filesystem

📦 REQUIREMENTS

• devlog CLI tool installed (https://github.com/jellydn/devlog)
• devlog-host native messaging host registered
• tmux (for server log capture)

🔒 PRIVACY

All data stays on your local machine. No analytics, no tracking, no external servers. Open source at https://github.com/jellydn/devlog

📚 DOCUMENTATION

Visit https://github.com/jellydn/devlog for complete setup instructions and configuration examples.

🐛 SUPPORT

Report issues at https://github.com/jellydn/devlog/issues
```

**Category**: Developer Tools

**Language**: English

**Homepage URL**: https://github.com/jellydn/devlog

**Support URL**: https://github.com/jellydn/devlog/issues

### Firefox Store Listing

**Name**: devlog - Browser Log Capture

**Summary** (250 characters max):
```
Zero-code local development log capture. Captures browser console logs from configured URLs and writes them to local files via native messaging host. Perfect for debugging local development environments.
```

**Description**:
```
devlog is a developer tool that captures browser console logs and writes them to local log files for debugging. Perfect for local development workflows where you need persistent logs.

FEATURES

• Zero code changes - no SDK or instrumentation required
• Captures console.log, console.error, console.warn, console.info, console.debug
• Captures uncaught errors and unhandled promise rejections
• URL pattern filtering - only capture logs from specific URLs
• Timestamped log entries
• Works with tmux sessions for comprehensive dev logging
• Completely local - no data sent to external servers

HOW IT WORKS

1. Configure URL patterns in devlog.yml
2. Extension captures console logs from matching pages
3. Logs are sent via Native Messaging to devlog-host
4. Logs are written to local files on your filesystem

REQUIREMENTS

• devlog CLI tool installed (https://github.com/jellydn/devlog)
• devlog-host native messaging host registered
• tmux (for server log capture)

PRIVACY

All data stays on your local machine. No analytics, no tracking, no external servers. Open source at https://github.com/jellydn/devlog

DOCUMENTATION

Visit https://github.com/jellydn/devlog for complete setup instructions and configuration examples.

SUPPORT

Report issues at https://github.com/jellydn/devlog/issues
```

**Version Release Notes**:
```
Initial release:
- Browser console log capture
- Native messaging integration
- URL pattern filtering
- Support for all console levels and errors
```

**Categories**: 
- Developer Tools
- Other

**Support Email**: Add your support email here

**Support Website**: https://github.com/jellydn/devlog/issues

**Homepage**: https://github.com/jellydn/devlog

**License**: MIT

## Version Management

When releasing new versions:

1. Update version in both manifest files:
   - `browser-extension/chrome/manifest.json`
   - `browser-extension/firefox/manifest.json`

2. Package both extensions:
   ```bash
   ./scripts/package-chrome.sh
   ./scripts/package-firefox.sh
   ```

3. Submit updated packages to both stores

4. Create a GitHub release tag:
   ```bash
   git tag -a extension-v1.0.1 -m "Browser extension v1.0.1"
   git push origin extension-v1.0.1
   ```

## Maintenance Checklist

After publishing:

- [ ] Monitor user reviews and ratings
- [ ] Respond to user feedback and issues
- [ ] Update store listings when features are added
- [ ] Keep manifest permissions minimal
- [ ] Test on latest browser versions before updates
- [ ] Maintain privacy policy compliance
- [ ] Update screenshots if UI changes significantly

## Common Issues

### Chrome Web Store

**Issue**: Extension rejected for permissions
- **Solution**: Provide detailed justification for each permission in the review notes

**Issue**: Screenshots not accepted
- **Solution**: Ensure dimensions are exactly 1280x800 or 640x400

**Issue**: Privacy policy required
- **Solution**: Link to PRIVACY.md in GitHub repository

### Firefox Add-ons

**Issue**: Source code review taking long
- **Solution**: Provide clear build instructions and repository link

**Issue**: Permission warnings
- **Solution**: Firefox shows warnings for powerful permissions. Document why they're needed.

**Issue**: Manifest V2 deprecation
- **Solution**: Firefox still supports Manifest V2. Plan migration to V3 in future.

## Resources

- [Chrome Web Store Developer Documentation](https://developer.chrome.com/docs/webstore/)
- [Firefox Add-on Developer Guide](https://extensionworkshop.com/)
- [Chrome Extension Manifest V3](https://developer.chrome.com/docs/extensions/mv3/)
- [Firefox Extension Manifest](https://developer.mozilla.org/en-US/docs/Mozilla/Add-ons/WebExtensions/manifest.json)
