import { readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, it, expect } from "vitest";
import { createChromeMock } from "./mocks/chrome.js";
import vm from "node:vm";

const __dirname = dirname(fileURLToPath(import.meta.url));
const source = readFileSync(resolve(__dirname, "../background.js"), "utf8");

function loadBackground(chromeOptions = {}) {
	const chrome = createChromeMock(chromeOptions);
	// Optional APIs referenced at load time
	chrome.action = undefined;
	chrome.browserAction = undefined;
	chrome.tabs = {
		query(_q, cb) {
			cb([]);
		},
		sendMessage() {},
	};
	const consoleLogs = [];
	const sandbox = {
		chrome,
		console: {
			log: (...a) => consoleLogs.push(["log", ...a]),
			warn: (...a) => consoleLogs.push(["warn", ...a]),
			error: (...a) => consoleLogs.push(["error", ...a]),
		},
	};
	vm.createContext(sandbox);
	vm.runInContext(source, sandbox, { filename: "background.js" });
	return { chrome, sandbox, consoleLogs };
}

function getHandler(chrome) {
	if (!chrome._listeners.onMessage.length) {
		throw new Error("no onMessage listener registered");
	}
	return chrome._listeners.onMessage[0];
}

describe("background.js", () => {
	it("does not auto-connect on load", () => {
		const { chrome } = loadBackground();
		expect(chrome._lastNativeHost).toBeUndefined();
	});

	it("GET_STATUS reports disconnected until first log", () => {
		const { chrome } = loadBackground();
		const handler = getHandler(chrome);
		let status;
		handler({ type: "GET_STATUS" }, {}, (resp) => {
			status = resp;
		});
		expect(status.connected).toBe(false);
		expect(status.enabled).toBe(true);
	});

	it("GET_CONFIG returns enabled flag based on URL patterns", () => {
		const { chrome } = loadBackground();
		const handler = getHandler(chrome);
		let resp;
		handler({ type: "GET_CONFIG", url: "http://localhost:3000/" }, {}, (r) => {
			resp = r;
		});
		expect(resp.enabled).toBe(true);
		expect(Array.isArray(resp.levels)).toBe(true);

		handler({ type: "GET_CONFIG", url: "https://evil.example/" }, {}, (r) => {
			resp = r;
		});
		expect(resp.enabled).toBe(false);
	});

	it("connects to native host and forwards LOG messages", () => {
		const { chrome } = loadBackground();
		const handler = getHandler(chrome);
		handler(
			{
				type: "LOG",
				level: "error",
				message: "boom",
				url: "http://localhost:3000/",
				timestamp: new Date().toISOString(),
			},
			{},
			() => {},
		);
		expect(chrome._lastNativeHost).toBe("com.devlog.host");
		expect(chrome._nativeMessages.length).toBeGreaterThanOrEqual(1);
		const msg = chrome._nativeMessages[chrome._nativeMessages.length - 1];
		expect(msg.level).toBe("error");
		expect(msg.message).toBe("boom");
	});

	it("marks disconnected after native port disconnect", () => {
		const { chrome } = loadBackground();
		const handler = getHandler(chrome);
		// Establish connection via LOG
		handler(
			{
				type: "LOG",
				level: "log",
				message: "hi",
				url: "http://localhost:3000/",
				timestamp: new Date().toISOString(),
			},
			{},
			() => {},
		);
		// Disconnect
		chrome._listeners.onDisconnect.forEach((fn) => fn());
		let status;
		handler({ type: "GET_STATUS" }, {}, (resp) => {
			status = resp;
		});
		expect(status.connected).toBe(false);
	});
});
