package app

import "github.com/heyrmi/testground/internal/challenge"

func otpInput() challenge.Challenge {
	return challenge.Challenge{
		ID:       "otp-input",
		Title:    "Six boxes pretending to be one field",
		URL:      "/app/otp-input",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "B. Awkward Inputs",
		Summary: "A one-time-code input built from six separate single-character boxes that " +
			"advance as you type, walk backwards on backspace, and distribute a pasted code " +
			"across themselves.",
		WhyHard: "It looks like one field and is six. Filling the first box with the whole code " +
			"leaves a single digit, because each box keeps only the last character it was " +
			"given -- so the obvious approach produces a value that is wrong in a way that " +
			"looks like a truncation bug rather than a test bug. Typing into each box in " +
			"turn fights the focus, since every keystroke moves it on before the next one " +
			"arrives. And the assembled value lives nowhere: there is no element holding " +
			"the code, only six holding one character each.",
		Hint: "Type one character per box and let the focus move itself, or paste the whole " +
			"code, which the paste handler spreads across the boxes in one step. Do not " +
			"select a box and fill it with six characters. Assert on the assembled value " +
			"the page publishes rather than trying to concatenate the boxes yourself, and " +
			"remember backspace on an empty box moves backwards rather than deleting.",
		Tags:     []string{"inputs", "otp", "focus", "paste"},
		Concepts: []string{"one field made of many", "focus moving between keystrokes", "paste as a bulk path", "no element holds the whole value"},
		Selectors: []challenge.Selector{
			{TestID: "otp-group", Note: "The row of boxes"},
			{TestID: "otp-0", Role: "textbox", Note: "First box; each is otp-<index>, zero based"},
			{TestID: "otp-5", Role: "textbox", Note: "Last box"},
			{TestID: "otp-value", Note: "The assembled code, which no input element holds"},
			{TestID: "otp-verdict", Note: "incomplete, accepted or rejected"},
			{TestID: "otp-clear", Role: "button", Note: "Empties every box"},
			{TestID: "expected-code", Note: "The code that will be accepted, printed on purpose"},
		},
		Stability: challenge.Stable,
	}
}

func fakeControls() challenge.Challenge {
	return challenge.Challenge{
		ID:       "fake-controls",
		Title:    "Controls that are not the elements they look like",
		URL:      "/app/fake-controls",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "B. Awkward Inputs",
		Summary: "A switch that is a div, a star rating that changes under the pointer, and a " +
			"slider with no input behind it that only a drag can move.",
		WhyHard: "None of these has the element a test reaches for. The switch has no checkbox, " +
			"so there is nothing to check and no checked property to read -- its state is an " +
			"attribute. The rating renders the hovered value rather than the chosen one, so " +
			"reading the stars while the pointer is still over them measures the pointer. " +
			"The slider has no input at all: no value to set, no arrow keys to press, and " +
			"the position is the state, so moving it means a real pointer sequence.",
		Hint: "Read state from where it lives rather than from where it would live in a native " +
			"control: the switch publishes aria-checked and a data attribute, and the page " +
			"prints the chosen rating separately from the shown one. For the rating, move " +
			"the pointer away before asserting. For the slider, press, move and release " +
			"with pointer events -- mouse events alone will not do it, and there is no value " +
			"to set instead.",
		Tags:     []string{"inputs", "aria", "hover", "pointer", "drag"},
		Concepts: []string{"state without a native control", "hover state versus chosen state", "pointer sequences", "reading aria for truth"},
		Selectors: []challenge.Selector{
			{TestID: "toggle", Role: "switch", Note: "A div with a role; its state is on data-state and aria-checked"},
			{TestID: "toggle-state", Note: "on or off, in text"},
			{TestID: "rating", Role: "radiogroup", Note: "The star row"},
			{TestID: "star-3", Role: "radio", Note: "Third star; each is star-<n>, one based"},
			{TestID: "rating-value", Note: "The rating actually chosen"},
			{TestID: "rating-shown", Note: "What the stars are currently drawing, which the pointer changes"},
			{TestID: "slider-track", Note: "Press and drag along this; there is no input"},
			{TestID: "slider-thumb", Note: "Follows the pointer while a button is held"},
			{TestID: "slider-value", Note: "Zero to a hundred, derived from the thumb's position"},
		},
		Stability: challenge.Stable,
	}
}
