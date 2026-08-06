package app

import "github.com/heyrmi/testground/internal/challenge"

func internationalisation() challenge.Challenge {
	return challenge.Challenge{
		ID:       "internationalisation",
		Title:    "The same page in five scripts",
		URL:      "/app/internationalisation",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T2,
		Category: "U. Internationalisation",
		Summary: "English, German, Arabic, Japanese and Hindi over the same content, with the " +
			"direction, the number and date formats, the currency and the plural rules all " +
			"following the choice. Beside them, two strings that look identical and compare " +
			"as different, and one emoji that is four code points.",
		WhyHard: "Almost every assertion here breaks under translation, and none of it looks " +
			"like a bug. A test that matches on English text fails in every other locale " +
			"while the feature works perfectly. A number assertion written as 1,234,567.89 " +
			"fails in German, where the separators swap, and a date written as 04/03/2026 " +
			"means two different days depending on who reads it. Arabic reverses the " +
			"layout, so anything positional inverts. German runs about a third longer, so " +
			"a button that fitted stops fitting. And the two strings that compare unequal " +
			"are the quiet one: they render identically, so a failure reads as the page " +
			"being wrong rather than the comparison.",
		Hint: "Assert on identifiers rather than on prose: the panel publishes its locale and " +
			"direction as attributes, and the formatted values are separate targets from " +
			"the words around them. Where you must compare text, normalise it first -- the " +
			"page shows the same comparison before and after, and only one of them is " +
			"true. Count graphemes rather than string length when the answer needs to " +
			"match what a person would say. And when a translated label overflows, that is " +
			"a finding rather than a locator problem.",
		Tags:     []string{"i18n", "rtl", "unicode", "formatting", "locale"},
		Concepts: []string{"prose assertions break under translation", "locale-dependent number and date formats", "normalisation before comparison", "graphemes are not characters"},
		Selectors: []challenge.Selector{
			{TestID: "language-switcher", Note: "The locale buttons"},
			{TestID: "locale-ar-EG", Role: "button", Note: "Switches to Arabic, which reverses the layout"},
			{TestID: "locale-panel", Note: "Publishes the current locale and direction as attributes"},
			{TestID: "greeting", Note: "Prose, and therefore the wrong thing to assert on"},
			{TestID: "translated-action", Role: "button", Note: "The label that grows under translation"},
			{TestID: "label-length", Note: "How long that label is in the current locale"},
			{TestID: "format-number", Note: "The same amount, formatted for the locale"},
			{TestID: "format-currency", Note: "The same amount as money"},
			{TestID: "format-date", Note: "The same instant, formatted short"},
			{TestID: "plural-one", Note: "Which plural category one falls in"},
			{TestID: "plural-two", Note: "Which category two falls in; not every language has two"},
			{TestID: "nfc", Note: "Composed form"},
			{TestID: "nfd", Note: "Decomposed form; identical on screen"},
			{TestID: "naive-equal", Note: "Whether they compare equal as written"},
			{TestID: "normalised-equal", Note: "Whether they compare equal once normalised"},
			{TestID: "family", Note: "One grapheme built from several code points"},
			{TestID: "family-length", Note: "Its string length, which is not one"},
			{TestID: "family-codepoints", Note: "Its code point count, which is also not one"},
			{TestID: "script-input", Role: "textbox", Note: "Takes the current script, and reports what came back"},
			{TestID: "typed-back", Note: "What the input round-tripped"},
			{TestID: "typed-length", Note: "The string length of that"},
		},
		Stability: challenge.Stable,
	}
}
