# Privacy Policy for devlog Browser Extension

**Last Updated:** February 14, 2026

## Overview

The devlog browser extension ("the Extension") is a developer tool designed to capture browser console logs for local development purposes. This privacy policy explains what data is collected, how it is used, and your rights regarding that data.

## Data Collection

### What Data is Collected

The Extension collects the following data **only from web pages that match the configured URL patterns** in your `devlog.yml` configuration file:

- Browser console logs (console.log, console.error, console.warn, console.info, console.debug)
- Uncaught JavaScript errors and stack traces
- Unhandled promise rejections
- Timestamp information for each log entry
- Log level/severity information

### What Data is NOT Collected

The Extension does **NOT** collect:

- Personal information (names, email addresses, etc.)
- Browsing history or visited URLs beyond the configured patterns
- Cookies or authentication tokens
- Form data or user inputs
- Network requests or responses
- Any data from web pages that don't match your configured URL patterns

## Data Usage

### Local Storage Only

All collected data is:

- **Stored locally** on your computer in log files specified in your `devlog.yml` configuration
- **Never transmitted** to any external servers or third-party services
- **Never shared** with the extension developers or any other parties
- **Completely under your control** - you can view, modify, or delete the log files at any time

### How the Extension Works

1. The Extension monitors browser console output on pages matching your configured URL patterns
2. Log messages are sent via Native Messaging protocol to the `devlog-host` native application running on your computer
3. The native host writes logs to local files on your filesystem
4. All data remains on your local machine

## Data Storage Location

Log files are stored in the location you specify in your `devlog.yml` configuration file, typically in your project directory. The default structure is:

```text
logs/
  browser/console.log
```

You have full control over:

- Where logs are stored
- How long logs are kept
- What URL patterns are monitored
- What log levels are captured

## Permissions Explanation

The Extension requires the following permissions:

- **`nativeMessaging`**: Required to communicate with the local `devlog-host` native application that writes logs to disk
- **`storage`**: Used to store your extension configuration (URL patterns, log levels) locally in the browser
- **`activeTab`**: Allows the extension to read console logs from the currently active browser tab
- **`<all_urls>`** (host permissions): Required to inject the log capture script into pages matching your configured URL patterns

These permissions are used **exclusively** for the functionality described above. No data is sent to external servers.

## User Control

You have complete control over the Extension:

- **Configuration**: You define which URLs are monitored via your `devlog.yml` file
- **Enable/Disable**: You can enable or disable the extension at any time through your browser's extension settings
- **Uninstall**: Removing the extension will stop all log collection immediately
- **Data Deletion**: You can delete log files at any time by removing them from your filesystem

## Third-Party Services

The Extension does **NOT**:

- Use any analytics or tracking services
- Connect to any external APIs or servers
- Share data with third parties
- Include advertisements or monetization

## Open Source

The Extension is open source software. You can review the complete source code at:
https://github.com/jellydn/devlog

This transparency allows you to verify exactly what the Extension does with your data.

## Children's Privacy

The Extension is a developer tool intended for use by software developers. It is not designed for or directed at children under the age of 13.

## Changes to This Policy

We may update this privacy policy from time to time. Any changes will be reflected in the "Last Updated" date at the top of this document. Continued use of the Extension after changes constitutes acceptance of the updated policy.

## Contact

If you have questions or concerns about this privacy policy, please:

- Open an issue on GitHub: https://github.com/jellydn/devlog/issues
- Email: dung@productsway.com

## Your Rights

Depending on your location, you may have certain rights regarding your data:

- **Access**: You can access all log data by viewing the log files on your computer
- **Deletion**: You can delete all data by removing the log files
- **Portability**: All logs are stored as plain text files that you can move or share as needed
- **Transparency**: The open source nature of the Extension provides complete transparency

## Data Security

Since all data is stored locally on your computer:

- Data security is managed by your computer's operating system and file system permissions
- No data is transmitted over networks (except locally via Native Messaging)
- You are responsible for securing your log files using appropriate file system permissions

## Consent

By installing and using the devlog Extension, you consent to the data collection and usage practices described in this privacy policy.

---

**License:** The devlog Extension is released under the MIT License.
**Copyright:** © 2026 Dung Huynh Duc
