document.addEventListener("DOMContentLoaded", () => {
	const statusEl = document.getElementById("status");
	const urlsEl = document.getElementById("urls");
	const footerEl = document.getElementById("footer");
	const toggleEl = document.getElementById("toggle");

	function render(response) {
		if (chrome.runtime.lastError || !response) {
			statusEl.textContent = "Unable to connect";
			statusEl.className = "status disabled";
			footerEl.textContent = "Extension error. Check console for details.";
			if (toggleEl) toggleEl.hidden = true;
			return;
		}

		if (toggleEl) {
			toggleEl.hidden = false;
			toggleEl.textContent = response.enabled ? "Pause capture" : "Resume capture";
		}

		if (response.enabled && response.connected) {
			statusEl.textContent = "Browser logging active";
			statusEl.className = "status enabled";
			footerEl.textContent = "Native host connected";
		} else if (response.enabled) {
			statusEl.textContent = "Native host not connected";
			statusEl.className = "status disabled";
			footerEl.textContent = "Run: devlog register";
			const helpLink = document.createElement("a");
			helpLink.href =
				"https://github.com/jellydn/devlog/blob/main/browser-extension/README.md";
			helpLink.textContent = "See setup guide";
			helpLink.target = "_blank";
			helpLink.style.display = "block";
			helpLink.style.marginTop = "8px";
			helpLink.style.fontSize = "12px";
			footerEl.appendChild(document.createElement("br"));
			footerEl.appendChild(helpLink);
		} else {
			statusEl.textContent = "Browser logging disabled";
			statusEl.className = "status disabled";
			footerEl.textContent = "Logging is off";
		}

		if (response.urls && response.urls.length > 0) {
			const ul = document.createElement("ul");
			response.urls.forEach((url) => {
				const li = document.createElement("li");
				li.textContent = url;
				ul.appendChild(li);
			});
			urlsEl.innerHTML = "";
			urlsEl.appendChild(ul);
		}
	}

	function refresh() {
		chrome.runtime.sendMessage({ type: "GET_STATUS" }, render);
	}

	if (toggleEl) {
		toggleEl.addEventListener("click", () => {
			chrome.runtime.sendMessage({ type: "GET_STATUS" }, (response) => {
				if (chrome.runtime.lastError || !response) return;
				chrome.runtime.sendMessage(
					{ type: "SET_ENABLED", enabled: !response.enabled },
					refresh,
				);
			});
		});
	}

	refresh();
});
