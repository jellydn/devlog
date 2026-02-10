// popup.js - Handles the extension popup UI

document.addEventListener("DOMContentLoaded", () => {
	const statusEl = document.getElementById("status");
	const urlsEl = document.getElementById("urls");

	// Request current status from background script
	chrome.runtime.sendMessage(
		{ type: "GET_CONFIG", url: window.location.href },
		(response) => {
			if (response) {
				if (response.enabled) {
					statusEl.textContent = "Browser logging enabled";
					statusEl.className = "status enabled";
				} else {
					statusEl.textContent = "Browser logging disabled";
					statusEl.className = "status disabled";
				}

				// Display configured URLs
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
		},
	);
});
