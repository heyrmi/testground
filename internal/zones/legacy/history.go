package legacy

import "github.com/heyrmi/testground/internal/challenge"

func history() page {
	return simplePage(challenge.Challenge{
		ID:       "history",
		Title:    "pushState, replaceState and the back button",
		URL:      "/legacy/history",
		Zone:     challenge.ZoneLegacy,
		Tier:     challenge.T2,
		Category: "H. Windows, Tabs, Navigation",
		Summary: "Buttons that change the URL without loading anything: pushState adds a " +
			"history entry, replaceState overwrites one, and a hash link changes the " +
			"fragment. The page rebuilds itself from the URL on every back and forward.",
		WhyHard: "None of this is a navigation. No request is made, no load event fires, and " +
			"a wait for navigation sits there until it times out on a page that already " +
			"changed. Going back lands on a URL the server never served, so what you see " +
			"was reconstructed by script from the address bar alone -- and if the script " +
			"is wrong, the URL and the page disagree while both look plausible. " +
			"replaceState leaves no entry behind, so back skips straight past it to " +
			"somewhere a test did not expect. The hash link never reaches the server at all.",
		Hint: "Wait for the page to reflect the new state rather than for a navigation that " +
			"is not going to happen. Assert on the URL and on the rendered state together: " +
			"agreeing is the contract, and each alone can be right while the pair is " +
			"wrong. Remember that going back once after a replaceState does not undo it.",
		Tags:     []string{"navigation", "history", "pushstate", "hash-routing"},
		Concepts: []string{"URL changes without navigation", "back reconstructs from the URL", "replaceState leaves no entry", "hash never reaches the server"},
		Selectors: []challenge.Selector{
			{TestID: "push-one", Role: "button", Note: "pushState to step 1; adds a history entry"},
			{TestID: "push-two", Role: "button", Note: "pushState to step 2; adds another"},
			{TestID: "replace", Role: "button", Note: "replaceState; changes the URL and adds nothing"},
			{TestID: "hash-link", Role: "link", Note: "Changes the fragment, which never reaches the server"},
			{TestID: "current-step", Note: "Rebuilt from the URL on every history move"},
			{TestID: "current-hash", Note: "The fragment currently in the address bar"},
			{TestID: "popstate-count", Note: "How many times the page rebuilt itself from history"},
		},
		Stability: challenge.Stable,
	})
}
