// background.js - Background script that manages native messaging connection
// and forwards logs from content scripts to the native host

// Native messaging host name (must match the name in the native host manifest)
const NATIVE_HOST_NAME = "com.devlog.host";

// Connection to native host
let nativePort = null;
let isNativeHostConnected = false;

// Configuration - auto-enabled with sensible defaults
let config = {
	enabled: true,
	urls: ["http://localhost:*/*", "http://127.0.0.1:*/*"],
	levels: ["error", "warn", "info", "log"],
	file: "browser.log",
};

// Connect to native messaging host
function connectToNativeHost() {
	try {
		nativePort = chrome.runtime.connectNative(NATIVE_HOST_NAME);
		isNativeHostConnected = true;
		console.log("devlog: Connected to native host");

		nativePort.onDisconnect.addListener(() => {
			if (chrome.runtime.lastError) {
				console.error(
					"devlog: Native host connection failed:",
					chrome.runtime.lastError.message,
				);
			}
			console.log("devlog: Disconnected from native host");
			isNativeHostConnected = false;
			nativePort = null;
		});

		nativePort.onMessage.addListener((message) => {
			// Handle messages from native host (acknowledgments)
			if (message.type === "ACK") {
				console.log("devlog: Received acknowledgment:", message.success);
			}
		});

		return true;
	} catch (error) {
		console.error(
			"devlog: Failed to connect to native host:",
			error.message || error,
		);
		if (chrome.runtime.lastError) {
			console.error("devlog: Runtime error:", chrome.runtime.lastError.message);
		}
		isNativeHostConnected = false;
		return false;
	}
}

// Disconnect from native host
function disconnectFromNativeHost() {
	if (nativePort) {
		nativePort.disconnect();
		nativePort = null;
		isNativeHostConnected = false;
		console.log("devlog: Disconnected from native host");
	}
}

// Send log message to native host
function sendToNativeHost(message) {
	if (!isNativeHostConnected || !nativePort) {
		// Try to reconnect if not connected
		if (!connectToNativeHost()) {
			console.warn(
				"devlog: Cannot send log - native host not connected. " +
					"Make sure to run 'devlog up' first to start the logging session.",
			);
			return false;
		}
	}

	try {
		nativePort.postMessage(message);
		return true;
	} catch (error) {
		console.error(
			"devlog: Failed to send message to native host:",
			error.message || error,
		);
		if (chrome.runtime.lastError) {
			const errorMsg = chrome.runtime.lastError.message || "";
			console.error("devlog: Runtime error:", errorMsg);
			if (errorMsg.includes("has exited")) {
				console.error(
					"devlog: The native host process has exited. " +
						"Make sure 'devlog up' is running and the native messaging host is registered.",
				);
			}
		}
		isNativeHostConnected = false;
		return false;
	}
}

// Check if URL matches configured patterns
function isUrlEnabled(url) {
	if (!config.enabled || !config.urls || config.urls.length === 0) {
		return false;
	}

	return config.urls.some((pattern) => {
		// Simple wildcard matching
		// Convert pattern to regex
		const regexPattern = pattern
			.replace(/[.+^${}()|[\]\\]/g, "\\$&") // Escape special chars
			.replace(/\*/g, ".*"); // Convert * to .*

		const regex = new RegExp(regexPattern);
		return regex.test(url);
	});
}

// Handle messages from content scripts
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
	if (message.type === "GET_STATUS") {
		sendResponse({
			enabled: config.enabled,
			connected: isNativeHostConnected,
			urls: config.urls,
			levels: config.levels,
		});
		return true;
	}

	if (message.type === "GET_CONFIG") {
		// Content script is requesting configuration
		const enabled = isUrlEnabled(message.url);
		sendResponse({
			enabled: enabled,
			levels: config.levels,
			urls: config.urls,
			hostName: NATIVE_HOST_NAME,
		});
		return true;
	}

	if (message.type === "CONTENT_SCRIPT_READY") {
		// Content script has loaded
		console.log("devlog: Content script ready for", message.url);
		sendResponse({ received: true });
		return true;
	}

	if (message.type === "LOG") {
		// Forward log to native host
		const success = sendToNativeHost({
			level: message.level,
			url: message.url,
			source: message.source,
			line: message.line,
			column: message.column,
			message: message.message,
			timestamp: message.timestamp,
		});
		sendResponse({ sent: success });
		return true;
	}

	if (message.type === "UPDATE_CONFIG") {
		// CLI is updating configuration
		config = { ...config, ...message.config };
		console.log("devlog: Configuration updated:", config);

		// Notify all tabs to update their config
		chrome.tabs.query({}, (tabs) => {
			tabs.forEach((tab) => {
				try {
					chrome.tabs.sendMessage(tab.id, { type: "CONFIG_UPDATED" }, () => {
						if (chrome.runtime.lastError) {
							// Ignore errors for tabs without content script
						}
					});
				} catch (e) {
					// Ignore
				}
			});
		});

		// Connect or disconnect based on enabled state
		if (config.enabled && !isNativeHostConnected) {
			connectToNativeHost();
		} else if (!config.enabled && isNativeHostConnected) {
			disconnectFromNativeHost();
		}

		sendResponse({ updated: true });
		return true;
	}

	return false;
});

// Note: We don't auto-connect on startup because:
// 1. The native host needs a log file path from devlog CLI
// 2. The user should run `devlog up` first to start logging sessions
// 3. Connection will be attempted when first log message is sent

// Listen for browser action click (Chrome Manifest V3)
if (chrome.action) {
	chrome.action.onClicked.addListener((tab) => {
		// Toggle logging for current tab
		console.log("devlog: Toggle logging for", tab.url);
	});
}

// Listen for browser action click (Firefox Manifest V2)
if (chrome.browserAction) {
	chrome.browserAction.onClicked.addListener((tab) => {
		// Toggle logging for current tab
		console.log("devlog: Toggle logging for", tab.url);
	});
}
