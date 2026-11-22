// Light/dark mode toggle.
//
// Flips Bulma's color scheme by setting data-theme on the <html> element
// (see https://bulma.io/documentation/features/dark-mode/). The vendored
// bundle ships [data-theme=light] and [data-theme=dark] scheme blocks, so a
// single attribute recolors the whole page with no per-component work.
//
// The attribute is always set to "light" or "dark" (never removed) so the
// choice overrides the system's prefers-color-scheme in both directions: a
// system set to dark still opens light until the switch says otherwise. The
// choice is persisted in localStorage and applied before first paint, so a
// returning dark-mode visitor never sees a flash of the light layout.
//
// It is a file rather than an inline handler so that an application adding a
// Content-Security-Policy of script-src 'self' keeps a working toggle; the
// theme must not require inline script. Everything here is enhancement: with
// JavaScript disabled the bundle falls back to prefers-color-scheme and every
// page still renders, only the manual switch is inert.
//
// Portable: no framework, no build step, no external assets. The only DOM
// contract is an element with id "wa-theme-toggle" (a checkbox input); its
// checked state is the source of truth for the theme.
(function () {
	"use strict";

	var STORAGE_KEY = "wa-theme";

	function applyTheme(dark) {
		document.documentElement.setAttribute("data-theme", dark ? "dark" : "light");
	}

	function storedTheme() {
		try {
			return localStorage.getItem(STORAGE_KEY) === "dark";
		} catch (err) {
			// Storage may throw in private mode or when disabled; a failure to
			// read is simply "no choice yet", which is light.
			return false;
		}
	}

	// Apply the persisted theme as early as possible. The script is loaded in
	// <head> without defer precisely so this runs before the body paints.
	applyTheme(storedTheme());

	function init() {
		var toggle = document.getElementById("wa-theme-toggle");
		if (!toggle) {
			// A page with no toggle (the login page carries no navbar) has
			// nothing to wire; the early apply above already set the scheme.
			return;
		}

		toggle.checked = storedTheme();

		toggle.addEventListener("change", function () {
			applyTheme(toggle.checked);
			try {
				localStorage.setItem(STORAGE_KEY, toggle.checked ? "dark" : "light");
			} catch (err) {
				// Storage unavailable; the theme still applies for this page.
			}
		});
	}

	if (document.readyState === "loading") {
		document.addEventListener("DOMContentLoaded", init);
	} else {
		init();
	}
})();
