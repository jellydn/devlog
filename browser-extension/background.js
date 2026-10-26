/**
 * Background service worker for Devlog browser extension.
 * Native port + config from devlog-host. Capture stays off for URLs that
 * do not match the patterns from devlog.yml.
 */

const NATIVE_HOST_NAME = "com.devlog.host";
const RECONNECT_DELAY_MS = 5000;

let nativePort = null;
let connectBlockedUntil = 0;
let config = {
  enabled: true,
  urls: ["http://localhost:*/*", "http://127.0.0.1:*/*"],
  levels: ["log", "warn", "error", "info", "debug", "trace"],
};

restoreEnabled();

function restoreEnabled() {
  try {
    if (!chrome.storage || !chrome.storage.local) {
      return;
    }
    chrome.storage.local.get("devlogEnabled", (items) => {
      if (chrome.runtime.lastError) {
        return;
      }
      if (typeof items.devlogEnabled === "boolean") {
        config.enabled = items.devlogEnabled;
      }
    });
  } catch (err) {
    console.error("Devlog: failed to read enabled flag", err);
  }
}

function isUrlEnabled(url) {
  if (!config.enabled || !url) return false;

  for (const pattern of config.urls) {
    const regexPattern = pattern
      .replace(/[.+?^${}()|[\]\\]/g, "\\$&")
      .replace(/\*/g, ".*");
    try {
      if (new RegExp("^" + regexPattern + "$").test(url)) {
        return true;
      }
    } catch (err) {
      console.error("Devlog: bad url pattern", pattern, err);
    }
  }
  return false;
}

function applyHostConfig(message) {
  if (Array.isArray(message.urls)) {
    config.urls = message.urls;
  }
  if (Array.isArray(message.levels) && message.levels.length > 0) {
    config.levels = message.levels;
  }
  notifyConfig();
}

function notifyConfig() {
  if (!chrome.tabs || !chrome.tabs.query) return;
  chrome.tabs.query({}, (tabs) => {
    for (const tab of tabs || []) {
      if (tab.id === undefined) continue;
      try {
        chrome.tabs.sendMessage(tab.id, { type: "CONFIG_UPDATED" }, () => {
          if (chrome.runtime.lastError) {
            // Tab has no content script.
          }
        });
      } catch (err) {
        // Ignore tabs that cannot receive the message.
      }
    }
  });
}

function connectToNativeHost() {
  if (nativePort) return true;
  if (Date.now() < connectBlockedUntil) return false;

  try {
    nativePort = chrome.runtime.connectNative(NATIVE_HOST_NAME);
    nativePort.onMessage.addListener((message) => {
      if (message.type === "CONFIG") {
        applyHostConfig(message);
        return;
      }
      if (message.type === "ACK") {
        console.log("Devlog: message acknowledged", message);
      }
    });
    nativePort.onDisconnect.addListener(() => {
      const error = chrome.runtime.lastError;
      if (error) {
        console.log("Devlog: native host disconnected:", error.message);
      }
      nativePort = null;
      connectBlockedUntil = Date.now() + RECONNECT_DELAY_MS;
    });
    return true;
  } catch (err) {
    console.error("Devlog: failed to connect to native host:", err);
    nativePort = null;
    connectBlockedUntil = Date.now() + RECONNECT_DELAY_MS;
    return false;
  }
}

function disconnectFromNativeHost() {
  if (!nativePort) return;
  try {
    nativePort.disconnect();
  } catch (err) {
    console.error("Devlog: error disconnecting:", err);
  }
  nativePort = null;
}

function sendLogToNativeHost(logEntry) {
  if (!config.enabled) return false;
  if (!isUrlEnabled(logEntry.url)) return false;
  if (!nativePort && !connectToNativeHost()) return false;

  try {
    nativePort.postMessage({
      level: logEntry.level,
      url: logEntry.url,
      source: logEntry.source,
      line: logEntry.line,
      column: logEntry.column,
      message: logEntry.message,
      timestamp: logEntry.timestamp,
    });
    return true;
  } catch (err) {
    console.error("Devlog: failed to send log:", err);
    nativePort = null;
    connectBlockedUntil = Date.now() + RECONNECT_DELAY_MS;
    return false;
  }
}

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === "GET_STATUS") {
    sendResponse({
      enabled: config.enabled,
      connected: Boolean(nativePort),
      urls: config.urls,
      levels: config.levels,
    });
    return true;
  }

  if (message.type === "LOG") {
    const sent = sendLogToNativeHost({
      level: message.level,
      message: message.message,
      url: message.url,
      source: message.source,
      line: message.line,
      column: message.column,
      timestamp: message.timestamp || new Date().toISOString(),
    });
    sendResponse({ sent });
    return true;
  }

  if (message.type === "GET_CONFIG") {
    sendResponse({
      enabled: isUrlEnabled(message.url),
      levels: config.levels,
      urls: config.urls,
    });
    return true;
  }

  if (message.type === "SET_ENABLED") {
    config.enabled = Boolean(message.enabled);
    try {
      if (chrome.storage && chrome.storage.local) {
        chrome.storage.local.set({ devlogEnabled: config.enabled });
      }
    } catch (err) {
      console.error("Devlog: failed to store enabled flag", err);
    }
    if (!config.enabled) {
      disconnectFromNativeHost();
    }
    notifyConfig();
    sendResponse({ enabled: config.enabled });
    return true;
  }

  return false;
});

chrome.runtime.onSuspend?.addListener(() => {
  disconnectFromNativeHost();
});
