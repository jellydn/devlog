document.addEventListener("DOMContentLoaded", () => {
	const statusEl = document.getElementById("status");
	const urlsEl = document.getElementById("urls");
	const footerEl = document.getElementById("footer");

	chrome.runtime.sendMessage({ type: "GET_STATUS" }, (response) => {
		if (chrome.runtime.lastError || !response) {
			statusEl.textContent = "Unable to connect";
			statusEl.className = "status disabled";
			if (footerEl) footerEl.textContent = "Extension error";
			return;
		}

		if (response.enabled && response.connected) {
			statusEl.textContent = "Browser logging active";
			statusEl.className = "status enabled";
			if (footerEl) footerEl.textContent = "Native host connected";
		} else if (response.enabled) {
			statusEl.textContent = "Enabled (host not connected)";
			statusEl.className = "status disabled";
			if (footerEl) footerEl.textContent = "Run: devlog register";
		} else {
			statusEl.textContent = "Browser logging disabled";
			statusEl.className = "status disabled";
			if (footerEl) footerEl.textContent = "Logging is off";
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
	});
});
