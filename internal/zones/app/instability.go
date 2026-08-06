package app

import "github.com/heyrmi/testground/internal/challenge"

func detachedElements() challenge.Challenge {
	return challenge.Challenge{
		ID:       "detached-elements",
		Title:    "Elements that leave between finding and using them",
		URL:      "/app/detached-elements",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "D. Dynamic Content & DOM Instability",
		Summary: "A list that discards and rebuilds every row on a timer, giving each one a new " +
			"DOM id each time. Beside it a button that removes itself after six hundred " +
			"milliseconds, and a field that unmounts mid-sentence.",
		WhyHard: "The rows are rebuilt rather than updated, so the element found a moment ago " +
			"is not the element on screen now -- it is detached, and acting on it fails " +
			"with a message about the element rather than about the rebuild that caused " +
			"it. Their DOM ids change every generation, so a selector built from one is " +
			"correct exactly until the next tick. The vanishing button and the unmounting " +
			"field are the same problem with a shorter fuse: the page moved on between the " +
			"decision to act and the act.",
		Hint: "Locate by something that survives the rebuild. Every row keeps a stable data " +
			"attribute while its id does not, and a locator that re-resolves on use rather " +
			"than a handle captured earlier is immune to the whole problem. If your tool " +
			"gives you handles, re-find rather than reuse. And when a thing is about to " +
			"leave, read what you need from it before doing anything slow.",
		Tags:     []string{"detachment", "re-render", "unstable-ids", "unmount"},
		Concepts: []string{"detached elements", "keys forcing remounts", "unstable DOM ids", "unmounting mid-interaction"},
		Selectors: []challenge.Selector{
			{TestID: "toggle-churn", Role: "button", Note: "Starts and stops the rebuilding; its state is on data-churning"},
			{TestID: "generation", Note: "How many times the list has been rebuilt"},
			{TestID: "unstable-row", Note: "One per row; data-name survives the rebuild, the id attribute does not"},
			{TestID: "row-dom-id", Note: "The id that row currently has, printed so the churn is visible"},
			{TestID: "row-action", Role: "button", Note: "Inside a row; records the row's name"},
			{TestID: "chosen", Note: "Which rows were successfully chosen"},
			{TestID: "summon", Role: "button", Note: "Makes the vanishing button appear"},
			{TestID: "vanishing", Role: "button", Transient: true, Note: "Removes itself after 600 ms"},
			{TestID: "vanish-clicks", Note: "How many times it was caught in time"},
			{TestID: "arm-unmount", Role: "button", Note: "Gives the field 800 ms to live"},
			{TestID: "doomed-field", Role: "textbox", Transient: true, Note: "Unmounts while you are typing into it"},
			{TestID: "form-gone", Transient: true, Note: "Replaces the field once it has gone"},
		},
		Controls: []challenge.Control{
			{Name: "churnMs", Kind: "query", Default: "400", Note: "Milliseconds between rebuilds, clamped to 50-10000."},
		},
		Stability: challenge.Stable,
	}
}

func modalPortal() challenge.Challenge {
	return challenge.Challenge{
		ID:       "modal-portal",
		Title:    "A modal that is not where you think it is",
		URL:      "/app/modal-portal",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "D. Dynamic Content & DOM Instability",
		Summary: "A dialog rendered through a portal onto the body rather than inside the " +
			"application root, with a hand-rolled focus trap, a scroll lock on the page " +
			"behind it, and an overlay that closes it when clicked directly.",
		WhyHard: "The dialog is not a descendant of the app root, so a locator scoped to the " +
			"component tree never finds it even though it is plainly on screen. The " +
			"background button is still in the DOM and still reported enabled while the " +
			"overlay sits over it, so the click is intercepted and the error names the " +
			"overlay rather than anything the test mentioned. Tab cannot leave the dialog, " +
			"so a test that tabs a fixed number of times ends up somewhere it did not " +
			"predict. And the page behind cannot scroll, which turns any scroll-into-view " +
			"on a background element into a silent no-op.",
		Hint: "Locate from the document rather than from the app root. Treat the intercepted " +
			"click as information: the overlay really is on top, and a user would have the " +
			"same problem. The scroll lock is published on the body as a data attribute, " +
			"so it can be asserted rather than inferred, and the dialog reports how it was " +
			"closed -- confirm, cancel, escape or overlay -- because after it closes there " +
			"is nothing left to ask.",
		Tags:     []string{"portal", "modal", "focus-trap", "scroll-lock", "overlay"},
		Concepts: []string{"portals escape the component tree", "intercepted clicks", "focus traps", "scroll locking"},
		Selectors: []challenge.Selector{
			{TestID: "open-modal", Role: "button", Note: "Opens the dialog"},
			{TestID: "background-button", Role: "button", Note: "Behind the overlay once the dialog is open"},
			{TestID: "background-clicks", Note: "How many clicks actually reached it"},
			{TestID: "modal-overlay", Transient: true, Note: "Child of body; clicking it directly closes the dialog"},
			{TestID: "modal", Role: "dialog", Transient: true, Note: "The dialog itself, also outside the app root"},
			{TestID: "modal-confirm", Role: "button", Transient: true, Note: "Inside the dialog; takes focus on open"},
			{TestID: "modal-cancel", Role: "button", Transient: true, Note: "Inside the dialog"},
			{TestID: "modal-outcome", Note: "confirmed, cancelled, escape or overlay"},
			{TestID: "scroll-state", Note: "locked or free, mirroring the body"},
		},
		Stability: challenge.Stable,
	}
}
