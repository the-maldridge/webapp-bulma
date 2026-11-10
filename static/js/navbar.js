// Reveal the navigation menu on narrow screens.
//
// Bulma's navbar collapses its menu below the touch breakpoint and expects the
// page to add a few lines that toggle the is-active class; it ships no JavaScript
// of its own. Without a toggle the menu is display:none on a phone and the
// navigation is simply gone, which is worse than an ugly menu, so the theme
// provides the missing lines.
//
// It is a file rather than an inline handler so that an application adding a
// Content-Security-Policy of script-src 'self' still has working navigation.
// The framework ships no policy because a useful one is specific to the
// application, and a theme that forced an inline handler would be the reason
// somebody weakens a policy they got right.
//
// Everything here is enhancement. The menu exists in the document whether or not
// this runs, so a browser with JavaScript disabled still has the links in its
// tab order and in its text output; the only thing lost is the disclosure widget.
(function () {
	"use strict";

	var burger = document.getElementById("wa-nav-toggle");
	var menu = document.getElementById("wa-nav-menu");
	if (!burger || !menu) {
		// A page with no menu has nothing to wire, which is a layout this script
		// was not written for rather than an error.
		return;
	}

	function setOpen(open) {
		menu.classList.toggle("is-active", open);
		burger.setAttribute("aria-expanded", String(open));
	}

	burger.addEventListener("click", function () {
		setOpen(burger.getAttribute("aria-expanded") !== "true");
	});

	// Escape closes the menu and returns focus to the button, because a disclosure
	// widget that can only be closed by clicking the thing that opened it leaves
	// keyboard users holding an open menu they cannot dismiss.
	document.addEventListener("keydown", function (event) {
		if (event.key === "Escape" && burger.getAttribute("aria-expanded") === "true") {
			setOpen(false);
			burger.focus();
		}
	});
})();
