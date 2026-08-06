package app

import "github.com/heyrmi/testground/internal/challenge"

func delayedElement() challenge.Challenge {
	return challenge.Challenge{
		ID:       "delayed-element",
		Title:    "Element that appears after a delay",
		URL:      "/app/delayed-element",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T1,
		Category: "C. Waits & Timing",
		Summary: "A message element is absent from the DOM for three seconds after the page " +
			"loads, then appears. The delay is fixed, never random.",
		WhyHard: "A locate that runs immediately finds nothing at all, so a framework that " +
			"throws on a missing element fails before the page is ready. Sleeping for a " +
			"guessed duration hides the problem on a fast laptop and surfaces it on a " +
			"loaded CI runner.",
		Hint: "Wait for the element's presence rather than for a duration. Most frameworks " +
			"retry a locate until a timeout expires, so asserting on the element directly " +
			"is usually enough; the thing to avoid is a fixed sleep.",
		Tags:     []string{"waits", "timing", "appearance"},
		Concepts: []string{"explicit wait", "polling for presence", "absence is not yet failure"},
		Selectors: []challenge.Selector{
			{TestID: "delay-pending", Note: "Placeholder shown while the wait runs"},
			{TestID: "delayed-message", Transient: true, Note: "Appears once the delay elapses; not in the DOM before that"},
			{TestID: "restart", Role: "button", Note: "Runs the wait again without reloading"},
			{TestID: "delay-ms", Note: "The delay currently in effect, in milliseconds"},
		},
		Controls: []challenge.Control{
			{
				Name:    "delayMs",
				Kind:    "query",
				Default: "3000",
				Note:    "Milliseconds before the element appears, clamped to 0-60000.",
			},
		},
		Stability: challenge.Stable,
	}
}
