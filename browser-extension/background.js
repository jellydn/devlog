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
	if (isNativeHostConnected && nativePort) {
		return true;
	}
	try {
		nativePort = chrome.runtime.connectNative(NATIVE_HOST_NAME);
		isNativeHostConnected = true;
		console.log("devlog: Connected to native host");

		nativePort.onDisconnect.addListener(() => {
			const err = chrome.runtime.lastError;
			console.log(
				"devlog: Disconnected from native host",
				err ? err.message : "",
			);
			isNativeHostConnected = false;
			nativePort = null;
		});

		nativePort.onMessage.addListener((message) => {
			if (message.type === "ACK") {
				console.log("devlog: Received acknowledgment:", message.success);
			}
		});

		return true;
	} catch (error) {
		console.error("devlog: Failed to connect to native host:", error);
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

// Force reconnect to pick up new wrapper/manifest path
function reconnectToNativeHost() {
	disconnectFromNativeHost();
	return connectToNativeHost();
}

// Send log message to native host
function sendToNativeHost(message) {
	if (!isNativeHostConnected || !nativePort) {
		if (!connectToNativeHost()) {
			console.warn("devlog: Cannot send log - native host not connected");
			return false;
		}
	}

	try {
		nativePort.postMessage(message);
		return true;
	} catch (error) {
		console.error("devlog: Failed to send message to native host:", error);
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
		const regexPattern = pattern
			.replace(/[.+^${}()|[\]\\]/g, "\\$&")
			.replace(/\*/g, ".*");

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
		console.log("devlog: Content script ready for", message.url);
		sendResponse({ received: true });
		return true;
	}

	if (message.type === "LOG") {
		console.log("devlog: LOG received", message.level, (message.message || "").substring(0, 60));
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
		config = { ...config, ...message.config };
		console.log("devlog: Configuration updated:", config);

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

		if (config.enabled && !isNativeHostConnected) {
			connectToNativeHost();
		} else if (!config.enabled && isNativeHostConnected) {
			disconnectFromNativeHost();
		}

		sendResponse({ updated: true });
		return true;
	}

	if (message.type === "RECONNECT") {
		console.log("devlog: Reconnect requested");
		const success = reconnectToNativeHost();
		sendResponse({ reconnected: success });
		return true;
	}

	return false;
});

// In MV3 service workers, use chrome.runtime.onStartup and onInstalled
// to initialize the native host connection reliably.
// For MV2 (Firefox), the immediate connect also works.
if (chrome.runtime.onStartup) {
	chrome.runtime.onStartup.addListener(() => {
		if (config.enabled) {
			connectToNativeHost();
		}
	});
}

if (chrome.runtime.onInstalled) {
	chrome.runtime.onInstalled.addListener(() => {
		if (config.enabled) {
			connectToNativeHost();
		}
	});
}

// Connect immediately for the initial load
if (config.enabled) {
	connectToNativeHost();
}

// Listen for browser action click (Chrome Manifest V3)
if (chrome.action) {
	chrome.action.onClicked.addListener((tab) => {
		console.log("devlog: Toggle logging for", tab.url);
	});
}

// Listen for browser action click (Firefox Manifest V2)
if (chrome.browserAction) {
	chrome.browserAction.onClicked.addListener((tab) => {
		console.log("devlog: Toggle logging for", tab.url);
	});
}
