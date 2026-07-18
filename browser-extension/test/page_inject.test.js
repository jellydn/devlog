import { readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, it, expect, beforeEach, vi } from "vitest";
import { JSDOM } from "jsdom";

const __dirname = dirname(fileURLToPath(import.meta.url));
const scriptSource = readFileSync(
	resolve(__dirname, "../page_inject.js"),
	"utf8",
);

function loadPageInject() {
	const dom = new JSDOM("<!doctype html><html><body></body></html>", {
		url: "http://localhost:3000/app",
		runScripts: "outside-only",
	});
	const { window } = dom;
	const messages = [];
	const originalPostMessage = window.postMessage.bind(window);
	window.postMessage = (data, origin) => {
		messages.push({ data, origin });
		return originalPostMessage(data, origin);
	};

	// Capture original console before inject wraps it.
	const origLog = window.console.log.bind(window.console);
	const logCalls = [];
	window.console.log = (...args) => {
		logCalls.push(args);
		return origLog(...args);
	};

	window.eval(scriptSource);
	return { window, messages, logCalls };
}

describe("page_inject.js", () => {
	it("guards against double injection", () => {
		const { window } = loadPageInject();
		expect(window.__devlogPageInjected).toBe(true);
		window.eval(scriptSource);
		// Still true; second eval returns early without throwing.
		expect(window.__devlogPageInjected).toBe(true);
	});

	it("preserves console.log behavior and postMessages a __devlog event", () => {
		const { window, messages } = loadPageInject();
		window.console.log("hello", 42);
		const devlog = messages.filter((m) => m.data && m.data.__devlog);
		expect(devlog.length).toBeGreaterThanOrEqual(1);
		const evt = devlog[devlog.length - 1].data;
		expect(evt.level).toBe("log");
		expect(evt.message).toContain("hello");
		expect(evt.message).toContain("42");
		expect(evt.stack).toBeTruthy();
		expect(evt.url).toContain("localhost:3000");
		expect(evt.timestamp).toBeTruthy();
	});

	it("serializes objects and falls back for circular refs", () => {
		const { window, messages } = loadPageInject();
		const circular = { a: 1 };
		circular.self = circular;
		window.console.error("obj", { ok: true }, circular);
		const evt = messages.filter((m) => m.data?.__devlog).pop().data;
		expect(evt.level).toBe("error");
		expect(evt.message).toContain('"ok":true');
		// circular becomes String(obj) fallback path or [object Object]
		expect(evt.message.length).toBeGreaterThan(0);
	});

	it("posts uncaught error events", () => {
		const { window, messages } = loadPageInject();
		const event = new window.ErrorEvent("error", {
			message: "boom",
			filename: "app.js",
			lineno: 10,
			colno: 5,
			error: null,
		});
		window.dispatchEvent(event);
		const evt = messages.filter((m) => m.data?.__devlog).pop().data;
		expect(evt.level).toBe("error");
		expect(evt.message).toContain("Uncaught Error: boom");
	});
});
