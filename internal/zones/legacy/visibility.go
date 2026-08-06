package legacy

import "github.com/heyrmi/testground/internal/challenge"

func visibility() page {
	return simplePage(challenge.Challenge{
		ID:       "visibility",
		Title:    "Six ways to be invisible, and they are not the same",
		URL:      "/legacy/visibility",
		Zone:     challenge.ZoneLegacy,
		Tier:     challenge.T3,
		Category: "C. Waits & Timing",
		Summary: "Seven buttons, all present in the DOM. One is display:none, one is " +
			"visibility:hidden, one is opacity:0, one has no size, one is positioned off " +
			"the screen, one sits under a transparent overlay, and one fades in over a " +
			"second. Every button that is actually clicked records itself.",
		WhyHard: "Every framework draws the line between visible and hidden somewhere " +
			"slightly different, and these seven land on both sides of every line. " +
			"opacity:0 is fully clickable and completely unreadable, so a click succeeds " +
			"against something no user can see. The off-screen one is laid out, has a size " +
			"and is reachable by script, while being unreachable by a person. The covered " +
			"one passes every visibility check there is, and the click lands on the " +
			"overlay -- the error names an element the test never mentioned. And the " +
			"fading one is visible and in place, except that during the transition it is " +
			"neither.",
		Hint: "Decide what you actually mean before choosing a check. If the question is " +
			"'can a user do this', the test is whether a real click reaches the element, " +
			"not whether a locator can find it. Click, then assert the element recorded " +
			"the click, and the six that lie to you separate themselves immediately. When " +
			"a click reports hitting something else, believe it: that overlay is a real " +
			"bug your users are hitting too.",
		Tags:     []string{"visibility", "waits", "actionability", "overlays", "transitions"},
		Concepts: []string{"visible is not one thing", "clickable is not the same as visible", "intercepted clicks", "transitions delay interactability"},
		Selectors: []challenge.Selector{
			{TestID: "btn-normal", Role: "button", Note: "The control, plainly visible and clickable"},
			{TestID: "btn-display-none", Role: "button", Note: "display:none — out of the layout entirely"},
			{TestID: "btn-visibility-hidden", Role: "button", Note: "visibility:hidden — occupies space, cannot be clicked"},
			{TestID: "btn-opacity-zero", Role: "button", Note: "opacity:0 — invisible and fully clickable"},
			{TestID: "btn-zero-size", Role: "button", Note: "Zero width and height"},
			{TestID: "btn-offscreen", Role: "button", Note: "Positioned far off the left edge"},
			{TestID: "btn-covered", Role: "button", Note: "Under a transparent overlay that takes the click"},
			{TestID: "overlay", Note: "The transparent overlay; this is what a click on btn-covered actually hits"},
			{TestID: "btn-fading", Role: "button", Note: "Fades and slides in over one second after Reveal"},
			{TestID: "reveal", Role: "button", Note: "Starts the fading button's transition"},
			{TestID: "clicked", Note: "Which buttons have recorded a real click"},
		},
		Stability: challenge.Stable,
	})
}
