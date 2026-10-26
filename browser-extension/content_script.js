// Content script. The page hook is injected only after GET_CONFIG says this
// frame's URL is enabled. Console lines from the few milliseconds before that
// callback are missed, because the hook is not installed yet. Messages that
// arrive after inject but before the callback are buffered.
(() => {
if (window.__devlogInjected) return;
window.__devlogInjected = true;

const currentUrl = window.location.href;
let isLoggingEnabled = false;
let configKnown = false;
let logLevels = ["log", "info", "warn", "error", "debug", "trace"];
const pageToken = Math.random().toString(36).slice(2) + Date.now().toString(36);
let injected = false;
const pending = [];

function parseStack(stack) {
  let source = "inline";
  let line = 0;
  let column = 0;
  if (!stack) {
    return { source, line, column };
  }
  const match = stack.match(
    /(?:at\s+.*\s*\(?(.*?):(\d+):(\d+)\)?|@(.*?):(\d+):(\d+))/,
  );
  if (!match) {
    return { source, line, column };
  }
  source = match[1] || match[4] || "inline";
  line = Number.parseInt(match[2] || match[5] || "0", 10) || 0;
  column = Number.parseInt(match[3] || match[6] || "0", 10) || 0;
  return { source, line, column };
}

function forwardLog(data) {
  if (!logLevels.includes(data.level)) return;
  const loc = parseStack(data.stack);
  try {
    chrome.runtime.sendMessage({
      type: "LOG",
      level: data.level,
      url: data.url,
      source: loc.source,
      line: loc.line,
      column: loc.column,
      message: data.message,
      timestamp: data.timestamp,
    });
  } catch (e) {
    console.error("devlog: Error sending message:", e);
  }
}

function flushPending() {
  const queued = pending.splice(0);
  for (const data of queued) {
    forwardLog(data);
  }
}

function injectPageScript() {
  if (injected) return;
  const root = document.documentElement;
  if (!root) return;
  injected = true;
  root.setAttribute("data-devlog-token", pageToken);
  try {
    const script = document.createElement("script");
    script.src = chrome.runtime.getURL("page_inject.js");
    script.onload = () => {
      script.remove();
      root.removeAttribute("data-devlog-token");
    };
    root.appendChild(script);
  } catch (e) {
    console.error("devlog: script injection failed:", e);
  }
}

function updateConfig() {
  try {
    chrome.runtime.sendMessage({ type: "GET_CONFIG", url: currentUrl }, (response) => {
      if (chrome.runtime.lastError || !response) {
        configKnown = true;
        return;
      }
      isLoggingEnabled = Boolean(response.enabled);
      if (response.levels) {
        logLevels = response.levels.map((level) => level.toLowerCase());
      }
      configKnown = true;
      if (isLoggingEnabled) {
        injectPageScript();
        flushPending();
      }
    });
  } catch (e) {
    console.error("devlog: GET_CONFIG threw:", e);
  }
}

window.addEventListener("message", (event) => {
  if (event.source !== window || !event.data || !event.data.__devlog) return;
  if (event.data.token !== pageToken) return;
  if (!configKnown) {
    pending.push(event.data);
    return;
  }
  if (!isLoggingEnabled) return;
  forwardLog(event.data);
});

updateConfig();

chrome.runtime.onMessage.addListener((message) => {
  if (message && message.type === "CONFIG_UPDATED") {
    updateConfig();
  }
});
})();
