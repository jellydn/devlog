// Minimal chrome.* stubs for unit tests.
export function createChromeMock(options = {}) {
	const listeners = {
		onMessage: [],
		onDisconnect: [],
		onNativeMessage: [],
	};
	const sent = [];
	const nativeMessages = [];

	const port = {
		postMessage(msg) {
			nativeMessages.push(msg);
		},
		disconnect() {
			listeners.onDisconnect.forEach((fn) => fn());
		},
		onDisconnect: {
			addListener(fn) {
				listeners.onDisconnect.push(fn);
			},
		},
		onMessage: {
			addListener(fn) {
				listeners.onNativeMessage.push(fn);
			},
		},
	};

	const chrome = {
		runtime: {
			lastError: null,
			getURL(path) {
				return `chrome-extension://test/${path}`;
			},
			connectNative(name) {
				if (options.connectNativeError) {
					throw new Error(options.connectNativeError);
				}
				chrome._lastNativeHost = name;
				return port;
			},
			sendMessage(message, cb) {
				sent.push(message);
				if (typeof cb === "function") {
					cb(options.sendMessageResponse ?? undefined);
				}
			},
			onMessage: {
				addListener(fn) {
					listeners.onMessage.push(fn);
				},
			},
		},
		storage: {
			local: {
				get(defaults, cb) {
					cb({ ...(defaults || {}), ...(options.storage || {}) });
				},
				set(_data, cb) {
					if (cb) cb();
				},
			},
		},
		_sent: sent,
		_nativeMessages: nativeMessages,
		_port: port,
		_listeners: listeners,
	};
	return chrome;
}
