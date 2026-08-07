package classic

import (
	"net/http"
	"sort"
	"strings"

	"github.com/heyrmi/testground/internal/challenge"
)

type fieldStateValues struct {
	// Arrived is the set of field names the server actually received, which is
	// the whole point of the page: three controls that look equally
	// uneditable, and only one of them is really absent from the request.
	Arrived  string
	Readonly string
	AriaOff  string
	Locked   string
}

func fieldStates() page {
	meta := challenge.Challenge{
		ID:       "field-states",
		Title:    "Readonly, disabled and aria-disabled",
		URL:      "/classic/field-states",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "A. Basic Controls",
		Summary: "Three fields that look uneditable in the same way and behave completely " +
			"differently, beside four fields labelled four different ways: a for " +
			"attribute, a wrapping label, an aria-label, and a placeholder pretending to " +
			"be one.",
		WhyHard: "readonly, disabled and aria-disabled are visually identical and diverge " +
			"everywhere else. readonly takes focus and posts its value; disabled does " +
			"neither and vanishes from the request entirely; aria-disabled does both while " +
			"telling assistive technology it does not. Tooling disagrees about that last " +
			"one: Playwright reads aria-disabled as disabled and refuses to type into the " +
			"field at all, while the browser lets any real user edit it freely -- so a " +
			"control annotated this way is one your tests cannot touch and your users can, " +
			"including the users who were told not to try. Labelling splits the same way: " +
			"a locator that finds a field by its visible text works for three of these and " +
			"fails for the placeholder, because a placeholder is not an accessible name.",
		Hint: "Submit the form and read which names arrived. That answers what each state " +
			"really does far more reliably than reading the styling. If your framework " +
			"refuses to interact with the aria-disabled field, treat the refusal as " +
			"information rather than an obstacle: the markup is wrong, not the test. For " +
			"the labels, " +
			"prefer locating by accessible name over locating by nearby text -- the field " +
			"it cannot find is the one with the accessibility defect, and noticing that is " +
			"the useful outcome, not a reason to switch to a positional selector.",
		Tags:     []string{"forms", "disabled", "readonly", "aria", "labels"},
		Concepts: []string{"disabled fields are not submitted", "aria-disabled is not disabled", "accessible names", "placeholder is not a label"},
		Selectors: []challenge.Selector{
			{TestID: "form", Note: "The form; what it posts is the difference between readonly and disabled"},
			{TestID: "field-readonly", Role: "textbox", Note: "readonly: focusable, uneditable, and still posted"},
			{TestID: "field-disabled", Role: "textbox", Note: "disabled: not focusable and never reaches the server"},
			{TestID: "field-aria-disabled", Role: "textbox", Note: "aria-disabled: editable in the browser and posted, but reported disabled by tooling"},
			{TestID: "field-labelled-for", Role: "textbox", Note: "Named by a label with a for attribute"},
			{TestID: "field-labelled-wrap", Role: "textbox", Note: "Named by the label wrapped around it"},
			{TestID: "field-labelled-aria", Role: "textbox", Note: "Named by aria-label, with no visible label"},
			{TestID: "field-unlabelled", Role: "textbox", Note: "Has only a placeholder, so it has no accessible name at all"},
			{TestID: "submit", Role: "button", Note: "Posts the form"},
			{TestID: "result-arrived", Transient: true, Note: "The field names that actually reached the server"},
			{TestID: "no-submission", Note: "Shown while nothing has been posted"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/field-states", Note: "Reports which field names arrived, answers 303"},
		},
		Stability: challenge.Stable,
	}

	return formPage(meta, func(r *http.Request) fieldStateValues {
		names := make([]string, 0, len(r.PostForm))
		for name := range r.PostForm {
			names = append(names, name)
		}
		sort.Strings(names)

		return fieldStateValues{
			Arrived:  strings.Join(names, ", "),
			Readonly: r.PostFormValue("readonly"),
			AriaOff:  r.PostFormValue("ariaDisabled"),
			Locked:   r.PostFormValue("locked"),
		}
	})
}
