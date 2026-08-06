package classic

import (
	"net/http"
	"strings"

	"github.com/heyrmi/testground/internal/challenge"
)

// textValues is what the form echoes back after the redirect.
type textValues struct {
	Text    string
	Email   string
	Number  string
	Tel     string
	URL     string
	Search  string
	Comment string
	// The password is deliberately not echoed. Its length is enough for a
	// test to prove the field was filled, and reflecting a submitted secret
	// back into the page is a habit worth not teaching.
	PasswordLength int
}

func textInputs() page {
	meta := challenge.Challenge{
		ID:       "text-inputs",
		Title:    "Text inputs and a full page reload",
		URL:      "/classic/text-inputs",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T1,
		Category: "A. Basic Controls",
		Summary: "Every text-flavoured input on one form: text, password, email, number, " +
			"tel, url, search and a textarea. Submitting posts the form and answers 303, " +
			"so the browser fetches the page again and the values come back from the " +
			"server rather than from the DOM you typed into.",
		WhyHard: "Nothing here is hostile, and that is the point: it is the baseline a suite " +
			"should pass before facing anything that is. The one real trap is the reload. " +
			"Every element reference held across the submit is stale afterwards, because " +
			"the elements it pointed at no longer exist -- the classic stale-element " +
			"failure, on a page simple enough to see exactly why it happens.",
		Hint: "Re-locate after the submit rather than reusing handles from before it. The " +
			"post answers 303 and the browser follows it automatically, so a test that " +
			"waits for the response of the POST is waiting for the wrong request; wait " +
			"for the page that follows. Pressing Enter in any of these fields submits the " +
			"form too, without touching the button.",
		Tags:     []string{"forms", "inputs", "post-redirect-get", "stale-elements"},
		Concepts: []string{"full page loads", "stale element references", "303 redirect after post", "Enter-key submission"},
		Selectors: []challenge.Selector{
			{TestID: "field-text", Role: "textbox", Note: "Plain text input"},
			{TestID: "field-password", Role: "textbox", Note: "Password input; its value is never echoed back"},
			{TestID: "field-email", Role: "textbox", Note: "type=email, so the browser validates it before posting"},
			{TestID: "field-number", Role: "spinbutton", Note: "type=number with min, max and step"},
			{TestID: "field-tel", Role: "textbox", Note: "type=tel, which the browser does not validate"},
			{TestID: "field-url", Role: "textbox", Note: "type=url, validated before posting"},
			{TestID: "field-search", Role: "searchbox", Note: "type=search, which some browsers give a clear button"},
			{TestID: "field-comment", Role: "textbox", Note: "Textarea with a maxlength of 200"},
			{TestID: "submit", Role: "button", Note: "Posts the form; Enter in any field does the same"},
			{TestID: "clear", Role: "button", Note: "Discards the recorded submission"},
			{TestID: "result", Transient: true, Note: "Table of what the server received; absent until something is posted"},
			{TestID: "submission-count", Transient: true, Note: "How many times this session has posted the form"},
			{TestID: "no-submission", Note: "Shown while nothing has been posted yet"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/text-inputs", Note: "Records the submission, answers 303 back to the page"},
			{Method: http.MethodPost, Path: "/classic/text-inputs/clear", Note: "Discards it, answers 303"},
		},
		Stability: challenge.Stable,
	}

	return formPage(meta, func(r *http.Request) textValues {
		field := func(name string) string { return strings.TrimSpace(r.PostFormValue(name)) }
		return textValues{
			Text:           field("text"),
			Email:          field("email"),
			Number:         field("number"),
			Tel:            field("tel"),
			URL:            field("url"),
			Search:         field("search"),
			Comment:        field("comment"),
			PasswordLength: len(r.PostFormValue("password")),
		}
	})
}
