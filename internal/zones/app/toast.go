package app

import "github.com/heyrmi/testground/internal/challenge"

func toast() challenge.Challenge {
	return challenge.Challenge{
		ID:       "toast",
		Title:    "Toast that appears and then removes itself",
		URL:      "/app/toast",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T2,
		Category: "C. Waits & Timing",
		Summary: "Each click raises a toast that removes itself from the DOM three seconds " +
			"later. Toasts stack, so clicking twice puts two on screen at once.",
		WhyHard: "The toast is rendered through a portal into document.body, so a locator " +
			"rooted at the application container never sees it. It is also gone by the " +
			"time a slow step finishes, and the failure reads as a missing element rather " +
			"than as a late assertion. Two quick clicks make a single test id match more " +
			"than one node.",
		Hint: "Locate from the document, not from the app root. Read what you need off the " +
			"toast before doing anything slow, and assert lasting facts against the " +
			"counters, which outlive the toast. When more than one can be on screen, " +
			"narrow by its sequence rather than by the shared test id.",
		Tags:     []string{"waits", "timing", "portal", "disappearing"},
		Concepts: []string{"portals", "transient elements", "strict locator matching", "durable assertion targets"},
		Selectors: []challenge.Selector{
			{TestID: "show-toast", Role: "button", Note: "Raises another toast"},
			{TestID: "toast-region", Note: "Portal container, a direct child of body rather than of the app root"},
			{TestID: "toast", Role: "status", Transient: true, Note: "Every visible toast carries this; more than one can match at a time"},
			{TestID: "toast-count", Note: "Toasts raised so far in this page load; survives dismissal"},
			{TestID: "toast-last", Note: "Sequence number of the most recently dismissed toast"},
			{TestID: "toast-live", Note: "How many toasts are on screen right now"},
			{TestID: "dismiss-ms", Note: "The dwell currently in effect, in milliseconds"},
		},
		Controls: []challenge.Control{
			{
				Name:    "dismissMs",
				Kind:    "query",
				Default: "3000",
				Note:    "Milliseconds a toast stays in the DOM, clamped to 100-60000.",
			},
		},
		Stability: challenge.Stable,
	}
}
