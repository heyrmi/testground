package wc

import "github.com/heyrmi/testground/internal/challenge"

func closedShadow() challenge.Challenge {
	return challenge.Challenge{
		ID:       "closed-shadow",
		Title:    "A closed shadow root, and what to do instead",
		URL:      "/wc/closed-shadow",
		Zone:     challenge.ZoneComponents,
		Tier:     challenge.T4,
		Category: "F. Shadow DOM & Web Components",
		Summary: "A component whose shadow root is closed, so nothing in the page can see " +
			"inside it. Beside it, an element that upgrades a second after the page is " +
			"ready, and one that exposes a styling hook through ::part.",
		WhyHard: "This is the hostile tier, and it is here to be recognised rather than " +
			"copied. A closed root makes shadowRoot null, so there is no traversal to " +
			"perform and no piercing to fall back on -- the elements inside are simply not " +
			"addressable, and a page-wide search returns nothing with no indication that " +
			"anything is being withheld. The late upgrade is the other trap: before it " +
			"runs, the element is unknown to the browser, has no shadow root and no " +
			"behaviour, and looks exactly like a component that failed to load rather than " +
			"one that has not arrived yet.",
		Hint: "Stop trying to reach inside; you cannot, and no selector will change that. Use " +
			"what the component exposes: a value property to read and write, a composed " +
			"event that still crosses the boundary, and a ::part hook for styling. If a " +
			"component you own has a closed root and no such surface, the finding is that " +
			"it needs one -- the test is telling you something true about the component. " +
			"For the late upgrade, wait for the upgraded marker rather than for the element, " +
			"which was there all along doing nothing.",
		Tags:     []string{"shadow-dom", "closed", "web-components", "upgrade", "part"},
		Concepts: []string{"closed roots are unaddressable", "properties and events as the supported surface", "custom element upgrade timing", "::part as a sanctioned hook"},
		Selectors: []challenge.Selector{
			{TestID: "closed-host", Note: "The closed component; its shadowRoot is null"},
			{TestID: "closed-escaped", Note: "In the light DOM; receives the composed event that crosses the boundary"},
			{TestID: "closed-read", Role: "button", Note: "Reads the component's value property into the page"},
			{TestID: "closed-write", Role: "button", Note: "Writes a value through the property, since the input cannot be reached"},
			{TestID: "closed-value", Note: "What the property last reported"},
			{TestID: "late-host", Note: "Unknown to the browser until it upgrades; data-upgraded appears then"},
			{TestID: "late-content", Note: "Inside the late element's open shadow root, which exists only after it upgrades"},
			{TestID: "part-host", Note: "Exposes a trigger part, styled from the page stylesheet"},
		},
		Stability: challenge.Stable,
	}
}
