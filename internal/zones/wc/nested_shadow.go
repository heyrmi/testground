package wc

import "github.com/heyrmi/testground/internal/challenge"

func nestedShadow() challenge.Challenge {
	return challenge.Challenge{
		ID:       "nested-shadow",
		Title:    "Three nested shadow roots",
		URL:      "/wc/nested-shadow",
		Zone:     challenge.ZoneComponents,
		Tier:     challenge.T3,
		Category: "F. Shadow DOM & Web Components",
		Summary: "An input and a button sit inside three open shadow roots, one within the " +
			"next. Submitting dispatches a composed custom event that crosses all three " +
			"boundaries and lands in the light DOM.",
		WhyHard: "document.querySelector cannot see into a shadow root, so a page-wide search " +
			"for the input returns nothing however specific the selector is. CSS " +
			"descendant combinators stop at the boundary too. Tools differ sharply here: " +
			"some pierce open roots for you, others require you to walk shadowRoot by " +
			"shadowRoot, and code written against one silently fails on the other.",
		Hint: "Enter each root explicitly: find the host element, step into its shadow root, " +
			"then repeat for the next host. All three roots here are open, so traversal " +
			"always works even where automatic piercing does not. The event the button " +
			"fires is composed, so the light-DOM echo can be asserted without traversing " +
			"anything at all.",
		Tags:     []string{"shadow-dom", "web-components", "nesting", "custom-events", "slots"},
		Concepts: []string{"open shadow roots", "shadow traversal", "slotted light DOM", "composed events"},
		Selectors: []challenge.Selector{
			{TestID: "shadow-host", Note: "Outermost custom element; the only one in the light DOM"},
			{TestID: "slotted-label", Note: "Light-DOM node projected into the outer root through a named slot"},
			{TestID: "inner-input", Role: "textbox", Note: "Three roots deep"},
			{TestID: "inner-submit", Role: "button", Note: "Three roots deep; fires the composed event"},
			{TestID: "inner-echo", Note: "Three roots deep; mirrors what was typed"},
			{TestID: "shadow-echo", Note: "In the light DOM; receives the value after it crosses every boundary"},
			{TestID: "shadow-submit-count", Note: "In the light DOM; counts events that escaped"},
		},
		Stability: challenge.Stable,
	}
}
