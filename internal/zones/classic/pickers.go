package classic

import (
	"net/http"

	"github.com/heyrmi/testground/internal/challenge"
)

type pickerValues struct {
	Volume   string
	Colour   string
	Date     string
	Time     string
	Moment   string
	Month    string
	Week     string
	Deadline string
}

func pickers() page {
	meta := challenge.Challenge{
		ID:       "pickers",
		Title:    "Sliders, colours and native date inputs",
		URL:      "/classic/pickers",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "A. Basic Controls",
		Summary: "The controls with no text to type into: a range slider, a colour input, " +
			"and the native date, time, datetime-local, month and week inputs. All of them " +
			"post an ordinary string.",
		WhyHard: "A slider has nothing to type into. It moves with the arrow keys, with a " +
			"drag, or by having its value set, and those three do not fire the same events " +
			"-- code that only sets the value can miss handlers that listen for input. The " +
			"colour input opens an operating-system dialog that no browser automation can " +
			"reach at all. Native date inputs render as separate segments in some engines " +
			"and accept a whole formatted string in others, so typing into them is " +
			"engine-dependent in a way that setting the value is not.",
		Hint: "For the slider, prefer the keyboard: focus it and press the arrow keys, which " +
			"produces the same events a person would. For the colour input there is no way " +
			"through the native dialog -- set the value and move on, and know that this is " +
			"a limitation rather than a technique. For the date inputs, the value format is " +
			"fixed by the HTML specification regardless of how the browser displays it: " +
			"yyyy-mm-dd, HH:MM, yyyy-Www.",
		Tags:     []string{"forms", "inputs", "slider", "date", "colour"},
		Concepts: []string{"keyboard interaction", "unreachable native dialogs", "value format versus displayed format", "events from setting a value"},
		Selectors: []challenge.Selector{
			{TestID: "field-volume", Role: "slider", Note: "range 0-100, step 10, starts at 30"},
			{TestID: "field-colour", Note: "type=color; clicking it opens a dialog automation cannot reach"},
			{TestID: "field-date", Note: "type=date; the value is always yyyy-mm-dd whatever the display shows"},
			{TestID: "field-time", Note: "type=time; the value is HH:MM"},
			{TestID: "field-moment", Note: "type=datetime-local"},
			{TestID: "field-month", Note: "type=month; the value is yyyy-mm"},
			{TestID: "field-week", Note: "type=week; the value is yyyy-Www"},
			{TestID: "field-deadline", Note: "type=date with min and max, so out-of-range values fail validation"},
			{TestID: "submit", Role: "button", Note: "Posts the form"},
			{TestID: "result", Transient: true, Note: "What the server received, absent until posted"},
			{TestID: "no-submission", Note: "Shown while nothing has been posted"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/pickers", Note: "Records the submission, answers 303"},
		},
		Stability: challenge.Stable,
	}

	return formPage(meta, func(r *http.Request) pickerValues {
		return pickerValues{
			Volume:   r.PostFormValue("volume"),
			Colour:   r.PostFormValue("colour"),
			Date:     r.PostFormValue("date"),
			Time:     r.PostFormValue("time"),
			Moment:   r.PostFormValue("moment"),
			Month:    r.PostFormValue("month"),
			Week:     r.PostFormValue("week"),
			Deadline: r.PostFormValue("deadline"),
		}
	})
}
