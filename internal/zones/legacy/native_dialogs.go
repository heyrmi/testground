package legacy

import "github.com/heyrmi/testground/internal/challenge"

func nativeDialogs() page {
	return simplePage(challenge.Challenge{
		ID:       "native-dialogs",
		Title:    "alert, confirm, prompt and the ones that stack",
		URL:      "/legacy/native-dialogs",
		Zone:     challenge.ZoneLegacy,
		Tier:     challenge.T2,
		Category: "G. Native Dialogs & Browser-Level",
		Summary: "Each of the three native dialogs, a pair that fires one straight after the " +
			"other, one that fires on a timer so it arrives without being asked for, and a " +
			"beforeunload prompt on the way out.",
		WhyHard: "A native dialog is not in the DOM. There is no element to locate, no text " +
			"to read with a selector, and nothing to click -- it is a browser-level " +
			"interrupt that blocks JavaScript until it is answered. Frameworks that do not " +
			"answer it automatically simply hang, and frameworks that dismiss it " +
			"automatically silently answer 'cancel' for you, which changes what the page " +
			"does without any test saying so. The chained pair catches handlers that only " +
			"expect one. The timed one arrives when no test asked for a dialog at all, " +
			"which is how a suite that was passing starts hanging after an unrelated " +
			"change.",
		Hint: "Register the handler before the action that triggers the dialog, not after -- " +
			"the dialog blocks the page, so there is no 'after' until it is answered. " +
			"Decide explicitly whether to accept or dismiss rather than relying on your " +
			"framework's default, and read the message it carried: that is the only place " +
			"the text exists. For a chained pair the handler fires more than once, so it " +
			"has to be able to answer each one differently.",
		Tags:     []string{"dialogs", "alert", "confirm", "prompt", "beforeunload"},
		Concepts: []string{"dialogs are not DOM", "handlers registered before the trigger", "accept versus dismiss changes behaviour", "unexpected dialogs hang a suite"},
		Selectors: []challenge.Selector{
			{TestID: "fire-alert", Role: "button", Note: "Opens an alert"},
			{TestID: "fire-confirm", Role: "button", Note: "Opens a confirm; the answer decides what the page writes"},
			{TestID: "fire-prompt", Role: "button", Note: "Opens a prompt; whatever is typed lands in the result"},
			{TestID: "fire-chain", Role: "button", Note: "Fires an alert and then a confirm, from one click"},
			{TestID: "fire-delayed", Role: "button", Note: "Fires an alert two seconds later, when nothing is waiting for it"},
			{TestID: "leave-link", Role: "link", Note: "Navigating away raises a beforeunload prompt"},
			{TestID: "dialog-result", Note: "What the page did with the answer; the only lasting record"},
			{TestID: "dialog-count", Note: "How many dialogs have been answered"},
		},
		Stability: challenge.Stable,
	})
}
