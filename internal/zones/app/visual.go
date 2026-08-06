package app

import "github.com/heyrmi/testground/internal/challenge"

func visualRegression() challenge.Challenge {
	return challenge.Challenge{
		ID:       "visual-regression",
		Title:    "A block that looks the same every time, until you ask it not to",
		URL:      "/app/visual-regression",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T2,
		Category: "R. Visual Regression Targets",
		Summary: "A fixed-size block drawn with system fonts and fixed colours, so a capture " +
			"of it is the same on every run. A query parameter widens one element by a " +
			"single pixel, another lets a spinner animate, and a timestamp inside is " +
			"marked for masking.",
		WhyHard: "A visual comparison that never fails is indistinguishable from one that " +
			"cannot fail, and most of them are the second without anyone finding out. Web " +
			"fonts that arrive after the first paint, animations mid-frame and any clock " +
			"on the page all make a capture differ from itself, so the usual response is " +
			"to raise the tolerance until it stops complaining -- at which point a real " +
			"one-pixel regression passes too. This page is built so that a capture is " +
			"stable by construction, and then offers a difference small enough to prove " +
			"the comparison is awake.",
		Hint: "Take the baseline with the defaults, then take another with the diff turned on " +
			"and require it to fail. A comparison that passes both ways is not comparing " +
			"anything, and that check is worth more than any number of green runs. The " +
			"timestamp carries a masking attribute rather than being hidden, because a " +
			"comparison that cannot ignore regions needs telling and pretending the " +
			"element is absent teaches the wrong habit. Freeze the animation rather than " +
			"widening the tolerance to cover it.",
		Tags:     []string{"visual-regression", "screenshot", "masking", "animation"},
		Concepts: []string{"a comparison that never fails may be unable to", "stability by construction", "masking volatile regions", "freezing rather than tolerating"},
		Selectors: []challenge.Selector{
			{TestID: "reference", Note: "The block to capture; fixed width and system fonts"},
			{TestID: "swatch", Note: "One pixel wider when diff is on"},
			{TestID: "volatile", Note: "Carries data-vr-mask; changes ten times a second"},
			{TestID: "spinner", Note: "Frozen at a fixed angle unless freeze is turned off"},
			{TestID: "diff-state", Note: "on or off"},
			{TestID: "freeze-state", Note: "frozen or running"},
		},
		Controls: []challenge.Control{
			{Name: "diff", Kind: "query", Default: "0", Note: "Set to 1 to widen the swatch by one pixel."},
			{Name: "freeze", Kind: "query", Default: "1", Note: "Set to 0 to let the spinner animate."},
		},
		Stability: challenge.Stable,
	}
}
