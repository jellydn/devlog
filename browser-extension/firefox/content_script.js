// content_script.js - Bridges between page-level console capture and the background script.
// Injects page_inject.js via web_accessible_resources (runs in page context, no CSP issues).
// Listens for postMessage from inject script and forwards to background.

(() => {
	if (window.__devlogInjected) return;
	window.__devlogInjected = true;

	function debug(message, err) {
		try {
			console.debug("[devlog] " + message, err || "");
		} catch (_) {}
	}

	const currentUrl = window.location.href;
	let isLoggingEnabled = false;
	let logLevels = ["log", "info", "warn", "error", "debug", "trace"];
	let messageCount = 0;

	function updateConfig() {
		try {
			chrome.runtime.sendMessage(
				{ type: "GET_CONFIG", url: currentUrl },
				(response) => {
					if (chrome.runtime.lastError) {
						debug(
							"GET_CONFIG failed:",
							chrome.runtime.lastError.message,
						);
						return;
					}
					if (response) {
						isLoggingEnabled = response.enabled;
						debug(
							"config updated: enabled=" +
								isLoggingEnabled +
								" url=" +
								currentUrl,
						);
						if (response.levels) {
							logLevels = response.levels.map((l) => l.toLowerCase());
						}
					} else {
						debug("GET_CONFIG returned empty response");
					}
				},
			);
		} catch (e) {
			debug("GET_CONFIG threw:", e);
		}
	}

	updateConfig();

	chrome.runtime.onMessage.addListener((message) => {
		if (message.type === "CONFIG_UPDATED") {
			updateConfig();
		}
	});

	// Inject page-level script into the MAIN world.
	// Always attempt createElement injection as a fallback — the guard
	// variable in page_inject.js prevents double-execution if the manifest
	// world:"MAIN" entry also loaded it.
	try {
		const script = document.createElement("script");
		script.src = chrome.runtime.getURL("page_inject.js");
		(document.documentElement || document.head || document.body).appendChild(script);
		script.onload = () => script.remove();
		debug("page_inject.js injected via createElement");
	} catch (e) {
		debug("script injection failed:", e);
	}

	// Listen for messages from injected page script
	window.addEventListener("message", (event) => {
		if (!event.data || event.data.__devlog !== true) return;
		if (typeof event.data.level !== "string" || typeof event.data.message !== "string") return;
		messageCount++;
		if (messageCount <= 3) {
			debug("postMessage received #" + messageCount + ": level=" + event.data.level + " enabled=" + isLoggingEnabled);
		}
		if (!isLoggingEnabled) {
			debug(
				"dropping log (not enabled): " +
					event.data.level +
					": " +
					(event.data.message || "").substring(0, 80),
			);
			return;
		}
		if (!logLevels.includes(event.data.level)) return;

		let source = "inline";
		let line = "0";
		let column = "0";

		if (event.data.stack) {
			const match = event.data.stack.match(
				/(?:at\s+.*\s*\(?(.*?):(\d+):(\d+)\)?|@(.*?):(\d+):(\d+))/,
			);
			if (match) {
				source = match[1] || match[4] || "inline";
				line = match[2] || match[5] || "0";
				column = match[3] || match[6] || "0";
			}
		}

		try {
			chrome.runtime.sendMessage(
				{
					type: "LOG",
					level: event.data.level,
					url: event.data.url,
					source: source,
					line: line,
					column: column,
					message: event.data.message,
					timestamp: event.data.timestamp,
				},
				() => {
					if (chrome.runtime.lastError) {
						debug(
							"LOG message failed:",
							chrome.runtime.lastError.message,
						);
					}
				},
			);
		} catch (e) {
			debug("LOG message threw:", e);
		}
	});

	try {
		chrome.runtime.sendMessage(
			{ type: "CONTENT_SCRIPT_READY", url: currentUrl },
			() => {
				if (chrome.runtime.lastError) {
					debug(
						"CONTENT_SCRIPT_READY failed:",
						chrome.runtime.lastError.message,
					);
				}
			},
		);
	} catch (e) {
		debug("CONTENT_SCRIPT_READY threw:", e);
	}
})();
