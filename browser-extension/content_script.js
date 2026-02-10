// content_script.js - Injected into all web pages to capture console logs
// This script runs in an isolated world but can access the page's DOM

(() => {
	// Check if we've already injected to avoid double-logging
	if (window.__devlogInjected) {
		return;
	}
	window.__devlogInjected = true;

	// Store original console methods
	const originalConsole = {
		log: console.log,
		info: console.info,
		warn: console.warn,
		error: console.error,
		debug: console.debug,
		trace: console.trace,
	};

	// Get current URL for filtering
	const currentUrl = window.location.href;

	// Check if logging is enabled for this URL
	let isLoggingEnabled = false;
	let logLevels = ["log", "info", "warn", "error", "debug", "trace"];
	let nativeHostName = "com.devlog.host";

	// Request configuration from background script
	function updateConfig() {
		chrome.runtime.sendMessage(
			{ type: "GET_CONFIG", url: currentUrl },
			(response) => {
				if (response) {
					isLoggingEnabled = response.enabled;
					if (response.levels) {
						logLevels = response.levels.map((l) => l.toLowerCase());
					}
					if (response.hostName) {
						nativeHostName = response.hostName;
					}
				}
			},
		);
	}

	// Initial config fetch
	updateConfig();

	// Listen for config updates from background
	chrome.runtime.onMessage.addListener((message) => {
		if (message.type === "CONFIG_UPDATED") {
			updateConfig();
		}
	});

	// Send log to background script (which forwards to native host)
	function sendLog(level, args, stack) {
		if (!isLoggingEnabled) {
			return;
		}

		// Check if this log level is enabled
		if (!logLevels.includes(level.toLowerCase())) {
			return;
		}

		// Format arguments
		const formattedArgs = Array.from(args).map((arg) => {
			if (typeof arg === "object") {
				try {
					return JSON.stringify(arg);
				} catch (e) {
					return String(arg);
				}
			}
			return String(arg);
		});

		// Build source location from stack trace
		let source = "";
		let line = "";
		let column = "";

		if (stack) {
			// Parse stack trace to get file:line:column
			const match = stack.match(/\s+at\s+.*\s*\(?(.*?):(\d+):(\d+)\)?/);
			if (match) {
				source = match[1];
				line = match[2];
				column = match[3];
			}
		}

		// Get the page URL at the time of the log
		const pageUrl = window.location.href;

		const logMessage = {
			type: "LOG",
			level: level,
			url: pageUrl,
			source: source || "inline",
			line: line || "0",
			column: column || "0",
			message: formattedArgs.join(" "),
			timestamp: new Date().toISOString(),
		};

		// Send to background script
		chrome.runtime.sendMessage(logMessage).catch(() => {
			// Ignore errors - native host might not be connected
		});
	}

	// Create wrapped console methods
	function createConsoleWrapper(level) {
		return (...args) => {
			// Call original console method
			originalConsole[level].apply(console, args);

			// Get stack trace
			const stack = new Error().stack;

			// Send to native host
			sendLog(level, args, stack);
		};
	}

	// Wrap console methods
	console.log = createConsoleWrapper("log");
	console.info = createConsoleWrapper("info");
	console.warn = createConsoleWrapper("warn");
	console.error = createConsoleWrapper("error");
	console.debug = createConsoleWrapper("debug");
	console.trace = createConsoleWrapper("trace");

	// Capture uncaught errors
	window.addEventListener("error", (event) => {
		const stack = event.error ? event.error.stack : "";
		sendLog(
			"error",
			[`Uncaught Error: ${event.message}`],
			stack || `at ${event.filename}:${event.lineno}:${event.colno}`,
		);
	});

	// Capture unhandled promise rejections
	window.addEventListener("unhandledrejection", (event) => {
		const reason = event.reason;
		let message = "Unhandled Promise Rejection";
		let stack = "";

		if (reason instanceof Error) {
			message = `Unhandled Promise Rejection: ${reason.message}`;
			stack = reason.stack;
		} else if (typeof reason === "string") {
			message = `Unhandled Promise Rejection: ${reason}`;
		} else {
			try {
				message = `Unhandled Promise Rejection: ${JSON.stringify(reason)}`;
			} catch (e) {
				message = `Unhandled Promise Rejection: ${String(reason)}`;
			}
		}

		sendLog("error", [message], stack);
	});

	// Notify that content script is ready
	chrome.runtime.sendMessage({ type: "CONTENT_SCRIPT_READY", url: currentUrl });
})();
