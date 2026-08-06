package legacy

import "github.com/heyrmi/testground/internal/challenge"

func dialogElement() page {
	return simplePage(challenge.Challenge{
		ID:       "dialog-element",
		Title:    "The dialog element, modal and not",
		URL:      "/legacy/dialog-element",
		Zone:     challenge.ZoneLegacy,
		Tier:     challenge.T2,
		Category: "G. Native Dialogs & Browser-Level",
		Summary: "Two dialog elements that look identical. One is opened with show and leaves " +
			"the page usable; the other is opened with showModal and makes everything " +
			"behind it inert. The modal one closes on Escape and carries a return value " +
			"back from a form.",
		WhyHard: "A modal dialog does not hide the page behind it. Every background element " +
			"is still in the DOM, still laid out, still reported visible and enabled -- and " +
			"the browser will not deliver a click to any of them. A locator finds the " +
			"button, every precondition passes, and the click is refused, which reads as a " +
			"flaky test rather than as the correct behaviour it is. The non-modal dialog " +
			"looks the same and does none of that. Escape closes the modal one and nothing " +
			"else, so dismissing out of habit closes a dialog nobody asked to close. And a " +
			"form with method=dialog posts nowhere: it closes the dialog and sets a return " +
			"value, which is the only record of which button was used.",
		Hint: "Check which kind you are looking at before deciding what should work: a modal " +
			"blocks the background, and a background click being refused is the feature. " +
			"The return value is the thing to assert on after a modal closes, not the " +
			"dialog's own contents, which are gone. Escape is a real interaction here, so " +
			"use it deliberately rather than as a way to clear state.",
		Tags:     []string{"dialogs", "modal", "inert", "focus", "keyboard"},
		Concepts: []string{"modal dialogs make the background inert", "visible and enabled is not clickable", "escape as an interaction", "form method=dialog"},
		Selectors: []challenge.Selector{
			{TestID: "open-modal", Role: "button", Note: "Opens the dialog with showModal"},
			{TestID: "open-modeless", Role: "button", Note: "Opens the dialog with show, leaving the page usable"},
			{TestID: "background-button", Role: "button", Note: "Behind the dialog; clickable only while no modal is open"},
			{TestID: "background-clicks", Note: "How many times the background button was actually reached"},
			{TestID: "modal-dialog", Note: "The modal dialog element; present in the DOM whether open or not"},
			{TestID: "modeless-dialog", Note: "The non-modal dialog element"},
			{TestID: "confirm-dialog", Role: "button", Note: "Inside the modal; closes it with a return value"},
			{TestID: "cancel-dialog", Role: "button", Note: "Inside the modal; closes it with a different one"},
			{TestID: "dialog-return", Note: "The return value the modal closed with; the only lasting record"},
		},
		Stability: challenge.Stable,
	})
}
