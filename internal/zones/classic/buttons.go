package classic

import (
	"net/http"

	"github.com/heyrmi/testground/internal/challenge"
)

type buttonValues struct {
	Action string
	Draft  string
}

func buttons() page {
	meta := challenge.Challenge{
		ID:       "buttons",
		Title:    "Six things that look like buttons",
		URL:      "/classic/buttons",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T1,
		Category: "A. Basic Controls",
		Summary: "Two submit buttons that post different values under the same name, a " +
			"reset, an inert type=button, a disabled one, an anchor styled to look " +
			"identical, and a submit whose label is wrapped in a child element.",
		WhyHard: "Half of these are not buttons. The anchor navigates instead of submitting, " +
			"and in a zone with no JavaScript the type=button does nothing whatsoever -- " +
			"which is indistinguishable from a click that silently failed. The disabled " +
			"one swallows clicks without complaint. The two submits share a name and " +
			"differ only by value, so which one was pressed is the only thing that " +
			"distinguishes the requests they send.",
		Hint: "Locate by role. An anchor is a link and a button is a button, and that " +
			"difference tells you whether to expect navigation or a post. Assert a " +
			"disabled control is disabled rather than clicking it and hoping. When two " +
			"submits share a name, check the value that arrived at the server, not which " +
			"element you think you pressed.",
		Tags:     []string{"forms", "buttons", "links", "disabled"},
		Concepts: []string{"role over appearance", "submit button values", "inert controls", "click targets inside a control"},
		Selectors: []challenge.Selector{
			{TestID: "submit-save", Role: "button", Note: "type=submit, posts action=save"},
			{TestID: "submit-publish", Role: "button", Note: "type=submit, posts action=publish"},
			{TestID: "reset", Role: "button", Note: "type=reset, clears the field without posting"},
			{TestID: "inert", Role: "button", Note: "type=button with no script behind it, so it does nothing"},
			{TestID: "disabled", Role: "button", Note: "Disabled; clicks are swallowed"},
			{TestID: "link-button", Role: "link", Note: "An anchor styled as a button; navigates rather than posting"},
			{TestID: "submit-icon", Role: "button", Note: "Its label sits in a child span, so a click can land on the child"},
			{TestID: "field-draft", Role: "textbox", Note: "Something for reset to clear"},
			{TestID: "result", Transient: true, Note: "Which action reached the server, absent until posted"},
			{TestID: "no-submission", Note: "Shown while nothing has been posted"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/buttons", Note: "Records which submit button posted, answers 303"},
		},
		Stability: challenge.Stable,
	}

	return formPage(meta, func(r *http.Request) buttonValues {
		return buttonValues{
			Action: r.PostFormValue("action"),
			Draft:  r.PostFormValue("draft"),
		}
	})
}
