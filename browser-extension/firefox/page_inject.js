(function () {
	if (window.__devlogPageInjected) return;
	window.__devlogPageInjected = true;

	var orig = {
		log: console.log.bind(console),
		info: console.info.bind(console),
		warn: console.warn.bind(console),
		error: console.error.bind(console),
		debug: console.debug.bind(console),
		trace: console.trace.bind(console),
	};

	function wrap(level) {
		return function () {
			orig[level].apply(console, arguments);
			try {
				var args = [];
				for (var i = 0; i < arguments.length; i++) {
					try {
						var a = arguments[i];
						if (typeof a === "object") {
							try {
								args.push(JSON.stringify(a));
							} catch (e) {
								args.push(String(a));
							}
						} else {
							args.push(String(a));
						}
					} catch (e) {
						args.push("[unreadable]");
					}
				}
				window.postMessage(
					{
						__devlog: true,
						level: level,
						message: args.join(" "),
						stack: new Error().stack || "",
						url: window.location.href,
						timestamp: new Date().toISOString(),
					},
					"*",
				);
			} catch (e) {
				orig.error("[devlog] page_inject wrap failed:", e);
			}
		};
	}

	console.log = wrap("log");
	console.info = wrap("info");
	console.warn = wrap("warn");
	console.error = wrap("error");
	console.debug = wrap("debug");
	console.trace = wrap("trace");

	window.addEventListener("error", function (event) {
		window.postMessage(
			{
				__devlog: true,
				level: "error",
				message: "Uncaught Error: " + event.message,
				stack: event.error
					? event.error.stack
					: "at " +
						event.filename +
						":" +
						event.lineno +
						":" +
						event.colno,
				url: window.location.href,
				timestamp: new Date().toISOString(),
			},
			"*",
		);
	});

	window.addEventListener("unhandledrejection", function (event) {
		var reason = event.reason;
		var message = "Unhandled Promise Rejection";
		var stack = "";
		if (reason instanceof Error) {
			message += ": " + reason.message;
			stack = reason.stack || "";
		} else if (typeof reason === "string") {
			message += ": " + reason;
		} else {
			try {
				message += ": " + JSON.stringify(reason);
			} catch (e) {
				message += ": " + String(reason);
			}
		}
		window.postMessage(
			{
				__devlog: true,
				level: "error",
				message: message,
				stack: stack,
				url: window.location.href,
				timestamp: new Date().toISOString(),
			},
			"*",
		);
	});
})();
