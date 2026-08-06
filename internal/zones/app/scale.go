package app

import "github.com/heyrmi/testground/internal/challenge"

func domScale() challenge.Challenge {
	return challenge.Challenge{
		ID:       "dom-scale",
		Title:    "A page heavy enough to change how your tools behave",
		URL:      "/app/dom-scale",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "T. Performance & Scale",
		Summary: "Twenty thousand nodes on demand, five hundred event listeners, a main thread " +
			"blocked for three seconds, a layout thrash loop, and a leak that retains a " +
			"little more on every click. Every effect is reported as a number.",
		WhyHard: "None of this changes what the page says, so a test that only reads content " +
			"passes throughout and reports nothing. What changes is how long everything " +
			"takes: a locator that walks twenty thousand nodes is measurably slower than " +
			"one that does not, and a suite that was comfortably inside its timeout stops " +
			"being so on a page like this. The blocked thread is the sharpest edge, " +
			"because it stops your framework's waiting too -- a poll that runs in the page " +
			"cannot run while the page is not running, so a three-second block looks " +
			"exactly like a three-second network delay from outside and needs a completely " +
			"different fix. The leak is invisible by construction: nothing on screen ever " +
			"mentions it.",
		Hint: "Measure rather than assume. Time the same locator with the nodes built and " +
			"without, and the cost is a number rather than a feeling. Prefer a scoped " +
			"locator to a document-wide one here; the difference is small on a normal page " +
			"and large on this one. Treat the blocked thread as a distinct failure from a " +
			"slow response, because waiting harder fixes one and not the other. For the " +
			"leak, the page publishes a retained count -- in a real application nothing " +
			"would, which is the actual lesson.",
		Tags:     []string{"performance", "scale", "main-thread", "memory", "layout-thrash"},
		Concepts: []string{"cost without visible change", "a blocked thread stops your waiting too", "scoped locators on heavy pages", "leaks are invisible by construction"},
		Selectors: []challenge.Selector{
			{TestID: "build-nodes", Role: "button", Note: "Builds the node count from the query string"},
			{TestID: "attach-listeners", Role: "button", Note: "Attaches five hundred listeners"},
			{TestID: "block-thread", Role: "button", Note: "Blocks the main thread synchronously"},
			{TestID: "toggle-thrash", Role: "button", Note: "Forces a synchronous reflow every frame; state is on data-thrashing"},
			{TestID: "leak", Role: "button", Note: "Retains another block that is never released"},
			{TestID: "node-host", Note: "Where the nodes are built; scope locators to it"},
			{TestID: "node-count", Note: "How many nodes exist"},
			{TestID: "listener-count", Note: "How many listeners are attached"},
			{TestID: "thread-state", Note: "free or blocked"},
			{TestID: "retained-count", Note: "How many blocks are retained; a real leak would report nothing"},
		},
		Controls: []challenge.Control{
			{Name: "nodes", Kind: "query", Default: "20000", Note: "Nodes to build, clamped to 0-60000."},
			{Name: "blockMs", Kind: "query", Default: "3000", Note: "Milliseconds to block the main thread, clamped to 0-20000."},
		},
		Stability: challenge.Stable,
	}
}
