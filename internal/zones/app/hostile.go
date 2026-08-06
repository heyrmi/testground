package app

import "github.com/heyrmi/testground/internal/challenge"

func hostileLocators() challenge.Challenge {
	return challenge.Challenge{
		ID:       "hostile-locators",
		Title:    "A page that gives you nothing to hold on to",
		URL:      "/app/hostile-locators",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T4,
		Category: "S. Hostile Locators",
		Summary: "Generated class names that change on every build, two elements sharing one " +
			"id, twelve levels of div with no semantics at the bottom, text split across " +
			"nodes, invisible characters, CSS truncation, and a pair of buttons identical " +
			"in every respect but position.",
		WhyHard: "Everything here is legal, everything here ships, and none of it is deliberate " +
			"sabotage on anyone's part -- which is why it is worth practising against. The " +
			"class names are content hashes, so a selector written against one is correct " +
			"until the next deploy and then fails with no code change anyone will connect " +
			"to it. The duplicate id is invalid HTML that browsers accept silently, so one " +
			"lookup finds the first and a CSS selector finds both. The split text reads as " +
			"one sentence and is three nodes, so an exact-match assertion fails against a " +
			"string the user can plainly see. The truncated line shows an ellipsis while " +
			"the DOM holds the whole thing, so what a person can read and what a test can " +
			"read have quietly diverged.",
		Hint: "This page is a diagnosis, not a puzzle. Every locator that survives it is one " +
			"anchored to something the application means rather than something it renders: " +
			"a test id, a role, an accessible name. Where none of those exist -- the div " +
			"soup, the twins -- the honest answer is that the markup needs fixing, and a " +
			"positional selector is a note to come back rather than a solution. For the " +
			"split and zero-width text, prefer a contains match over an exact one and " +
			"normalise before comparing.",
		Tags:     []string{"locators", "hostile", "css-in-js", "duplicate-ids", "text"},
		Concepts: []string{"generated class names are not selectors", "duplicate ids are legal and silent", "visible text is not DOM text", "positional selectors are a diagnosis"},
		Selectors: []challenge.Selector{
			{TestID: "rebuild", Role: "button", Note: "Regenerates every class name, as a deploy would"},
			{TestID: "build-number", Note: "Which build the class names belong to"},
			{TestID: "sample-class", Note: "One current class name, so the churn is observable"},
			{TestID: "chosen", Note: "Which element was last activated; the only stable target here"},
			{TestID: "split-text", Note: "Reads as one sentence, is three nodes"},
			{TestID: "zero-width", Note: "Contains zero-width spaces between the words"},
			{TestID: "truncated", Note: "Shows an ellipsis; the DOM holds the whole string"},
		},
		HostileLocators: true,
		Stability:       challenge.Stable,
	}
}
