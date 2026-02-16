# Browser Extension Publication Checklist

Use this checklist when publishing the devlog browser extension to Chrome Web Store and Firefox Add-ons.

## Pre-Submission Checklist

### Preparation

- [x] Privacy policy created ([PRIVACY.md](../PRIVACY.md))
- [x] Extension icons generated (16x16, 32x32, 48x48, 128x128)
- [x] Manifest files updated with store metadata
- [x] Packaging scripts created and tested
- [x] Store submission documentation created
- [ ] Screenshots captured (see [SCREENSHOTS.md](SCREENSHOTS.md))
- [ ] Promotional images created (optional)
- [ ] Test extension in both Chrome and Firefox
- [ ] Review all permissions and justifications

### Documentation Review

- [x] README.md updated with store information
- [x] PRIVACY.md reviewed for accuracy
- [x] Store listing copy prepared
- [x] Version number confirmed (currently 1.0.0)

## Chrome Web Store Submission

### Account Setup

- [ ] Create Chrome Web Store developer account
  - URL: https://chrome.google.com/webstore/devconsole/
  - Cost: $5 one-time fee
  - Payment method: Credit card
- [ ] Complete developer profile
- [ ] Agree to developer terms

### Package Extension

- [ ] Update version in `browser-extension/chrome/manifest.json` if needed
- [ ] Run packaging script:
  ```bash
  ./scripts/package-chrome.sh
  ```
- [ ] Verify package contents:
  ```bash
  unzip -l dist/devlog-chrome-v1.0.0.zip
  ```

### Store Listing

- [ ] Click "New Item" in developer dashboard
- [ ] Upload `dist/devlog-chrome-v1.0.0.zip`
- [ ] Fill in required fields:
  - [ ] Extension name: "devlog - Browser Log Capture"
  - [ ] Short description (132 chars max)
  - [ ] Detailed description (see [STORE_SUBMISSION.md](STORE_SUBMISSION.md))
  - [ ] Category: Developer Tools
  - [ ] Language: English
  - [ ] Homepage URL: https://github.com/jellydn/devlog
  - [ ] Support URL: https://github.com/jellydn/devlog/issues

### Assets Upload

- [ ] Store icon (128x128): `browser-extension/icons/icon128.png`
- [ ] Screenshots (at least 1, recommend 3-5):
  - [ ] Extension popup screenshot
  - [ ] Console capture screenshot
  - [ ] Configuration screenshot
  - [ ] Log output screenshot
- [ ] Promotional images (optional):
  - [ ] Small tile (440x280)
  - [ ] Marquee (1400x560)

### Privacy & Permissions

- [ ] Privacy policy URL: https://github.com/jellydn/devlog/blob/main/PRIVACY.md
- [ ] Single purpose description
- [ ] Permission justifications:
  - **nativeMessaging**: Communicates with local devlog-host binary to write logs
  - **storage**: Stores user configuration (URL patterns, log levels)
  - **activeTab**: Reads console logs from active browser tabs
  - **<all_urls>**: Required to inject content script on user-configured URLs

### Distribution

- [ ] Select distribution: Public
- [ ] Set visibility: Unlisted or Public (recommend Public)
- [ ] Set supported regions: All regions

### Submit

- [ ] Review all information
- [ ] Click "Submit for Review"
- [ ] Monitor submission status (typically 1-3 business days)
- [ ] Respond to any review feedback

### Post-Submission

- [ ] Check email for review updates
- [ ] Address any issues if rejected
- [ ] Once approved, verify listing is live
- [ ] Test installation from store
- [ ] Update README.md with store link
- [ ] Announce release

## Firefox Add-ons Submission

### Account Setup

- [ ] Create Firefox Add-ons developer account
  - URL: https://addons.mozilla.org/developers/
  - Cost: Free
- [ ] Complete developer profile
- [ ] Agree to developer terms

### Package Extension

- [ ] Update version in `browser-extension/firefox/manifest.json` if needed
- [ ] Run packaging script:
  ```bash
  ./scripts/package-firefox.sh
  ```
- [ ] Verify package contents:
  ```bash
  unzip -l dist/devlog-firefox-v1.0.0.zip
  ```

### Submit Add-on

- [ ] Click "Submit a New Add-on"
- [ ] Choose distribution: "On this site"
- [ ] Upload `dist/devlog-firefox-v1.0.0.zip`

### Source Code Review

Firefox requires source code review. Provide:

- [ ] Source code location: https://github.com/jellydn/devlog
- [ ] Build instructions:
  ```
  The extension requires no build step. All source files are included.
  To package manually, see browser-extension/README.md
  ```
- [ ] Dependencies: None (vanilla JavaScript)
- [ ] Additional notes: Extension uses native messaging to communicate with local devlog-host binary

### Listing Details

- [ ] Add-on name: "devlog - Browser Log Capture"
- [ ] Summary (250 chars max)
- [ ] Description (see [STORE_SUBMISSION.md](STORE_SUBMISSION.md))
- [ ] Version notes
- [ ] Categories:
  - [ ] Developer Tools
  - [ ] Other
- [ ] Tags: logging, developer, console, debugging, local-development
- [ ] Homepage: https://github.com/jellydn/devlog
- [ ] Support email: [Add your support email]
- [ ] Support website: https://github.com/jellydn/devlog/issues
- [ ] License: MIT

### Assets Upload

- [ ] Icon (128x128): `browser-extension/icons/icon128.png`
- [ ] Screenshots (at least 1):
  - [ ] Extension popup screenshot
  - [ ] Console capture screenshot
  - [ ] Configuration screenshot
  - [ ] Log output screenshot

### Privacy

- [ ] Privacy policy URL: https://github.com/jellydn/devlog/blob/main/PRIVACY.md
- [ ] Does extension collect user data: No
- [ ] Privacy practices description

### Technical Details

- [ ] Compatibility:
  - [ ] Firefox: 109+
  - [ ] Android: No (requires native host)
- [ ] Permissions explanations provided

### Submit

- [ ] Review all information
- [ ] Click "Submit Version"
- [ ] Monitor submission status (typically 1-5 business days for first submission)
- [ ] Respond to reviewer questions promptly

### Post-Submission

- [ ] Check email for review updates
- [ ] Be prepared to answer questions about source code
- [ ] Address any issues if rejected
- [ ] Once approved, verify listing is live
- [ ] Test installation from store
- [ ] Update README.md with store link
- [ ] Announce release

## Post-Publication Maintenance

### After Both Stores Approve

- [ ] Update README.md with store links:
  - Chrome Web Store: [Add link]
  - Firefox Add-ons: [Add link]
- [ ] Create GitHub release:
  ```bash
  git tag -a extension-v1.0.0 -m "Browser extension v1.0.0 - Initial store release"
  git push origin extension-v1.0.0
  ```
- [ ] Update browser-extension/README.md with store links
- [ ] Announce on social media / project channels
- [ ] Monitor reviews and ratings
- [ ] Set up process for handling user feedback

### Version Updates

When releasing new versions:

- [ ] Update version in both manifest.json files
- [ ] Update VERSION file
- [ ] Update browser-extension/README.md with changelog
- [ ] Package both extensions
- [ ] Upload new versions to both stores
- [ ] Provide clear version notes
- [ ] Create GitHub release tag
- [ ] Announce updates

### Monitoring

- [ ] Set up alerts for new reviews
- [ ] Respond to user reviews (especially negative ones)
- [ ] Monitor store analytics
- [ ] Track installation numbers
- [ ] Watch for policy changes from stores
- [ ] Keep permissions minimal
- [ ] Maintain privacy policy

## Support Resources

- [Chrome Web Store Developer Documentation](https://developer.chrome.com/docs/webstore/)
- [Firefox Extension Workshop](https://extensionworkshop.com/)
- [Store Submission Guide](STORE_SUBMISSION.md)
- [Screenshots Guide](SCREENSHOTS.md)
- [Privacy Policy](../PRIVACY.md)
- [Browser Extension README](../browser-extension/README.md)

## Common Issues & Solutions

### Chrome Web Store

**Issue**: "Extension violates policy"
- Review permissions and remove unnecessary ones
- Ensure privacy policy is accessible
- Provide detailed permission justifications

**Issue**: "Screenshots don't meet requirements"
- Resize to exactly 1280x800 or 640x400
- Ensure PNG or JPEG format
- Keep file size under 8 MB

### Firefox Add-ons

**Issue**: "Source code not clear"
- Provide detailed build instructions
- Link to exact GitHub commit/tag
- Explain any non-obvious code

**Issue**: "Permission concerns"
- Document why each permission is needed
- Show where in code each permission is used
- Consider removing unused permissions

## Notes

- Keep this checklist updated as you go through the process
- Document any store-specific quirks or issues encountered
- Store credentials and account information securely
- Set calendar reminders to check store dashboards monthly
