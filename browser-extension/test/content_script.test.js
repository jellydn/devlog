import { readFileSync } from "node:fs";
import { resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, it, expect } from "vitest";
import { JSDOM } from "jsdom";
import { createChromeMock } from "./mocks/chrome.js";

const __dirname = dirname(fileURLToPath(import.meta.url));
const source = readFileSync(resolve(__dirname, "../content_script.js"), "utf8");

function loadContentScript(chromeOptions = {}) {
	const chrome = createChromeMock({
		sendMessageResponse: { enabled: true, levels: ["log", "error"] },
		...chromeOptions,
	});
	const dom = new JSDOM("<!doctype html><html><body></body></html>", {
		url: "http://localhost:3000/",
		runScripts: "outside-only",
	});
	const { window } = dom;
	window.chrome = chrome;
	// content_script uses chrome as global
	const sandboxGlobals = { window, document: window.document, chrome, console: window.console };
	// Evaluate in window scope
	const script = new window.Function(
		"chrome",
		"window",
		"document",
		"console",
		source.replace(/^\(\(\) => \{/, "return (() => {"),
	);
	// The file is an IIFE; run it with our chrome
	window.eval(`var chrome = arguments[0];\n${source}`);
	// jsdom window.eval doesn't take args — set chrome on window and use with
	return { window, chrome };
}

function loadContent(chromeOptions = {}) {
	const chrome = createChromeMock({
		sendMessageResponse: { enabled: true, levels: ["log", "error", "warn"] },
		...chromeOptions,
	});
	const dom = new JSDOM("<!doctype html><html><body></body></html>", {
		url: "http://localhost:3000/",
		runScripts: "dangerously",
	});
	// Attach chrome before script runs
	dom.window.chrome = chrome;
	// Make chrome a free variable for the content script
	const wrapped = `
		var chrome = window.chrome;
		${source}
	`;
	dom.window.eval(wrapped);
	return { window: dom.window, chrome };
}

describe("content_script.js", () => {
	it("sets __devlogInjected guard and ignores second load", () => {
		const { window } = loadContent();
		expect(window.__devlogInjected).toBe(true);
		const chrome2 = createChromeMock({
			sendMessageResponse: { enabled: true, levels: ["log"] },
		});
		window.chrome = chrome2;
		window.eval(`var chrome = window.chrome;\n${source}`);
		// Second load should early-return without another CONTENT_SCRIPT_READY
		const readyMsgs = chrome2._sent.filter((m) => m.type === "CONTENT_SCRIPT_READY");
		expect(readyMsgs.length).toBe(0);
	});

	it("forwards __devlog postMessage as LOG when enabled", async () => {
		const { window, chrome } = loadContent();
		// Allow GET_CONFIG callback to apply
		await new Promise((r) => setTimeout(r, 0));

		window.dispatchEvent(
			new window.MessageEvent("message", {
				data: {
					__devlog: true,
					level: "error",
					message: "from page",
					url: "http://localhost:3000/",
					timestamp: new Date().toISOString(),
					stack: "Error\n    at app.js:1:1",
				},
				source: window,
			}),
		);

		const logs = chrome._sent.filter((m) => m.type === "LOG");
		expect(logs.length).toBe(1);
		expect(logs[0].level).toBe("error");
		expect(logs[0].message).toBe("from page");
	});

	it("does not forward when logging disabled", async () => {
		const { window, chrome } = loadContent({
			sendMessageResponse: { enabled: false, levels: ["error"] },
		});
		await new Promise((r) => setTimeout(r, 0));

		window.dispatchEvent(
			new window.MessageEvent("message", {
				data: {
					__devlog: true,
					level: "error",
					message: "nope",
					url: "http://localhost:3000/",
					timestamp: new Date().toISOString(),
				},
				source: window,
			}),
		);
		const logs = chrome._sent.filter((m) => m.type === "LOG");
		expect(logs.length).toBe(0);
	});

	it("filters by log level", async () => {
		const { window, chrome } = loadContent({
			sendMessageResponse: { enabled: true, levels: ["error"] },
		});
		await new Promise((r) => setTimeout(r, 0));

		window.dispatchEvent(
			new window.MessageEvent("message", {
				data: {
					__devlog: true,
					level: "log",
					message: "filtered",
					url: "http://localhost:3000/",
					timestamp: new Date().toISOString(),
				},
				source: window,
			}),
		);
		expect(chrome._sent.filter((m) => m.type === "LOG").length).toBe(0);
	});
});
