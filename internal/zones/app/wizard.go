package app

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

func wizard() challenge.Challenge {
	return challenge.Challenge{
		ID:       "wizard",
		Title:    "Four steps, a branch, and validation that runs late",
		URL:      "/app/wizard",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "V. Composite Scenarios",
		Summary: "Four steps collect an application: account type and email, then contact " +
			"details, then details whose boxes depend on the account type, then a review " +
			"that submits. Nothing is checked until Next is pressed, going back keeps what " +
			"was typed, and the step number lives in the page rather than in the address " +
			"bar. The server keeps its own copy of the draft, hears about a step only when " +
			"that step is validated and left forwards, and re-checks the whole thing itself " +
			"when the application is submitted.",
		WhyHard: "Next is enabled on an invalid step, so waiting for it to become enabled is " +
			"waiting for something that was already true, and the error a test wants to " +
			"assert on does not exist until after the click. The step lives in component " +
			"state and the URL never changes, so a reload restarts the wizard at step one " +
			"and the browser's back button leaves the flow entirely -- a test that reaches " +
			"step three and reloads to get back to a known state has quietly started " +
			"again, while the server's draft still holds everything the page has " +
			"forgotten. Step three asks an individual and a business different questions, " +
			"so a locator written against one branch finds nothing on the other and reports " +
			"a missing element rather than a wrong answer three steps back. Nothing " +
			"re-validates a step you did not revisit: change the account type on step one, " +
			"jump forward through the progress links, and the review shows a business " +
			"application while the server still holds an individual one, because the server " +
			"is only told about a step when Next validates it. And the email is checked " +
			"here for its shape and at the far end for its domain, so an address step one " +
			"accepted is refused by a message about a step you are no longer standing on.",
		Hint: "The Next button's state says nothing worth asserting on; assert on the error " +
			"that appears after the click. Ask the server what it has rather than believing " +
			"the review -- the draft endpoint is the authoritative copy, and it learns a " +
			"step only when that step is validated and left forwards, so a value still " +
			"sitting in a box is a value nobody has stored. Branch on the account type " +
			"rather than on the step number, because the page publishes which branch it is " +
			"showing. There is no step in the URL and so no shortcut into the middle of the " +
			"flow: a test that needs step three has to walk to step three. And every " +
			"refusal names the field and the step that produced it, so read where a failure " +
			"was caused instead of where it surfaced.",
		Tags:     []string{"composite", "wizard", "forms", "validation", "navigation"},
		Concepts: []string{"validation that runs on advance", "step in component state rather than the URL", "branching forms", "server as the source of truth", "failures that surface steps from their cause"},
		Selectors: []challenge.Selector{
			{TestID: "step-counter", Note: "Reads \"Step 2 of 4\"; the page's only record of where you are, because the URL never changes"},
			{TestID: "branch", Note: "unchosen, individual or business; step three shows the business boxes only when this reads business"},
			{TestID: "step-link", Role: "button", Note: "One per step; narrow by data-step. Enabled once the step before it has been cleared, and a jump through one validates nothing. aria-current marks the step you are on"},
			{TestID: "next", Role: "button", Note: "Enabled whether or not the step is valid; the errors arrive after the click"},
			{TestID: "back", Role: "button", Transient: true, Note: "Absent on the first step. Keeps what was typed in the page and tells the server nothing"},
			{TestID: "field-error", Transient: true, Note: "One per box the step just refused; narrow by data-field"},
			{TestID: "account-type", Role: "combobox", Note: "individual or business; decides what step three asks"},
			{TestID: "email", Role: "textbox", Note: "Checked here for its shape only; the domain is checked at submit"},
			{TestID: "full-name", Role: "textbox", Transient: true, Note: "Step two"},
			{TestID: "phone", Role: "textbox", Transient: true, Note: "Step two; seven digits or more, however they are spaced"},
			{TestID: "date-of-birth", Role: "textbox", Transient: true, Note: "Step three, individual branch only, as YYYY-MM-DD; the page checks the shape and the server checks the age"},
			{TestID: "occupation", Role: "textbox", Transient: true, Note: "Step three, individual branch only"},
			{TestID: "company-number", Role: "textbox", Transient: true, Note: "Step three, business branch only; exactly eight digits"},
			{TestID: "employees", Role: "spinbutton", Transient: true, Note: "Step three, business branch only"},
			{TestID: "review", Transient: true, Note: "Step four's summary, built from the page's copy of the draft rather than the server's"},
			{TestID: "review-value", Transient: true, Note: "One per reviewed field; narrow by data-field"},
			{TestID: "submit", Role: "button", Transient: true, Note: "Sends nothing; the server validates the draft it already holds"},
			{TestID: "submit-error", Transient: true, Note: "The server refused the application"},
			{TestID: "problem", Transient: true, Note: "One per refusal, carrying data-field and data-step: which box caused it and how far back that box was"},
			{TestID: "confirmation", Transient: true, Note: "Replaces the whole flow once an application is accepted"},
			{TestID: "reference", Transient: true, Note: "Exists nowhere until the confirmation renders"},
			{TestID: "start-again", Role: "button", Transient: true, Note: "Empties the page and returns to step one; the accepted application stays on the server"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/api/app/wizard/draft", Note: "What the server actually holds, which steps it has been told about, and every application accepted"},
			{Method: http.MethodPost, Path: "/api/app/wizard/draft", Note: "Records one step's boxes; a step may only write its own fields"},
			{Method: http.MethodPost, Path: "/api/app/wizard/submit", Note: "Validates the stored draft and ignores the request body entirely"},
		},
		Controls: []challenge.Control{
			{
				Name:    "clock",
				Kind:    "control-plane",
				Default: "real time",
				Note: "POST /api/control/clock moves the session clock, and the age on an " +
					"individual application is measured against it, so a date of birth can " +
					"be made old enough or too young without editing the date.",
			},
		},
		Stability: challenge.Stable,
	}
}

const (
	wizardStateKey        = "wizard"
	wizardSteps           = 4
	wizardMinimumAge      = 18
	wizardMinimumDigits   = 7
	wizardCompanyDigits   = 8
	wizardReferencePrefix = "WZ-"
	wizardDateLayout      = "2006-01-02"
)

// wizardStepOfField says which step owns each draft key, and the key is also
// the box's data-testid and the data-field on every error about it. One string
// for all three means a refusal can be traced from the server's problem list to
// the box that caused it without a translation table in the middle, which is
// the whole reason a refusal raised steps away from its cause is diagnosable.
var wizardStepOfField = map[string]int{
	"account-type":   1,
	"email":          1,
	"full-name":      2,
	"phone":          2,
	"date-of-birth":  3,
	"occupation":     3,
	"company-number": 3,
	"employees":      3,
}

// wizardRefusedDomains are published on the page rather than hidden. A test
// picks the path it is exercising instead of discovering which address happens
// to fail, the same way the checkout's card numbers work.
var wizardRefusedDomains = []string{"rejected.test", "disposable.test"}

// wizardDraft is one session's application in progress. It lives on the session
// rather than in a package variable so two workers filling in the same form at
// the same time never overwrite each other's answers.
type wizardDraft struct {
	mu     sync.Mutex
	values map[string]string
	// steps records which steps the client has validated and left forwards,
	// which is not the same as which steps it has shown. The gap between the two
	// is what this challenge is about, so it has to be readable.
	steps        map[int]bool
	applications []wizardApplication
}

// wizardApplication is a draft that passed. It exists only after a submit the
// server agreed with, and its reference is the only lasting record of it.
type wizardApplication struct {
	Reference   string            `json:"reference"`
	Values      map[string]string `json:"values"`
	SubmittedAt time.Time         `json:"submittedAt"`
}

// wizardProblem is one refusal. Field and Step name where the value came from
// rather than where the failure surfaced, because on a four-step form those are
// rarely the same place.
type wizardProblem struct {
	Field   string `json:"field"`
	Step    int    `json:"step"`
	Message string `json:"message"`
}

type wizardView struct {
	Values       map[string]string   `json:"values"`
	Steps        []int               `json:"steps"`
	Branch       string              `json:"branch"`
	Applications []wizardApplication `json:"applications"`
}

func wizardFor(sess *session.Session) *wizardDraft {
	return session.Value(sess, wizardStateKey, func() *wizardDraft {
		return &wizardDraft{values: map[string]string{}, steps: map[int]bool{}}
	})
}

// record stores one step's answers. A value is only accepted from the step that
// owns it, so a client posting the whole form under one step number cannot
// backfill a step it never rendered -- the server's draft stays a record of
// what was actually filled in and left, which is the thing worth asserting on.
func (d *wizardDraft) record(step int, values map[string]string) {
	if step < 1 || step > wizardSteps {
		return
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	for name, value := range values {
		if wizardStepOfField[name] == step {
			d.values[name] = strings.TrimSpace(value)
		}
	}
	d.steps[step] = true
}

func (d *wizardDraft) view() wizardView {
	d.mu.Lock()
	defer d.mu.Unlock()

	steps := make([]int, 0, len(d.steps))
	for step := range d.steps {
		steps = append(steps, step)
	}
	slices.Sort(steps)

	applications := make([]wizardApplication, len(d.applications))
	copy(applications, d.applications)

	return wizardView{
		Values:       maps.Clone(d.values),
		Steps:        steps,
		Branch:       wizardBranch(d.values),
		Applications: applications,
	}
}

// submit re-validates everything the server holds and turns it into an
// application. It reports had=false for a draft that is not there at all, which
// is a different answer from a draft that does not validate and deserves a
// different status: one means the flow was never walked, the other means it was
// walked wrongly.
func (d *wizardDraft) submit(now time.Time) (app wizardApplication, problems []wizardProblem, had bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.values) == 0 {
		return wizardApplication{}, nil, false
	}
	if problems := wizardProblemsIn(now, d.values); len(problems) > 0 {
		return wizardApplication{}, problems, true
	}

	app = wizardApplication{
		Reference:   fmt.Sprintf("%s%d", wizardReferencePrefix, 1000+len(d.applications)+1),
		Values:      maps.Clone(d.values),
		SubmittedAt: now,
	}
	d.applications = append(d.applications, app)

	// Accepting spends the draft it was made from, so a second submit from a
	// page still showing the review fails rather than lodging the application
	// twice.
	d.values = map[string]string{}
	d.steps = map[int]bool{}

	return app, nil, true
}

func wizardBranch(values map[string]string) string {
	switch values["account-type"] {
	case "individual", "business":
		return values["account-type"]
	default:
		return "unchosen"
	}
}

// wizardProblemsIn is the authoritative check, and it is deliberately stricter
// than the one the page makes. The page checks shapes and this checks meanings:
// an address is well formed there and at an acceptable domain here, a date of
// birth is well formed there and old enough here. Every one of those gaps is a
// refusal that arrives steps away from the box that caused it, which is the
// failure this page exists to teach.
//
// Answers belonging to the branch that was abandoned are ignored rather than
// deleted. A draft is a record of what was entered, and quietly discarding an
// answer because a later one changed is how a wizard loses work.
func wizardProblemsIn(now time.Time, values map[string]string) []wizardProblem {
	var problems []wizardProblem
	refuse := func(field, message string) {
		problems = append(problems, wizardProblem{
			Field:   field,
			Step:    wizardStepOfField[field],
			Message: message,
		})
	}

	branch := wizardBranch(values)
	if branch == "unchosen" {
		refuse("account-type", "choose whether this is an individual or a business")
	}

	domain, wellFormed := wizardEmailDomain(values["email"])
	switch {
	case !wellFormed:
		refuse("email", "that is not an email address")
	case slices.Contains(wizardRefusedDomains, domain):
		refuse("email", "we do not accept applications from "+domain)
	}

	if len([]rune(strings.TrimSpace(values["full-name"]))) < 2 {
		refuse("full-name", "a full name is required")
	}
	if wizardDigits(values["phone"]) < wizardMinimumDigits {
		refuse("phone", "a phone number needs at least seven digits")
	}

	switch branch {
	case "individual":
		born, err := time.Parse(wizardDateLayout, strings.TrimSpace(values["date-of-birth"]))
		switch {
		case err != nil:
			refuse("date-of-birth", "a date of birth is required, as YYYY-MM-DD")
		case born.AddDate(wizardMinimumAge, 0, 0).After(now):
			refuse("date-of-birth", "applicants must be eighteen or over")
		}
		if strings.TrimSpace(values["occupation"]) == "" {
			refuse("occupation", "an occupation is required")
		}
	case "business":
		company := strings.TrimSpace(values["company-number"])
		if len(company) != wizardCompanyDigits || wizardDigits(company) != wizardCompanyDigits {
			refuse("company-number", "a company number is exactly eight digits")
		}
		if count, err := strconv.Atoi(strings.TrimSpace(values["employees"])); err != nil || count < 1 {
			refuse("employees", "a business needs at least one employee")
		}
	}

	return problems
}

// wizardEmailDomain returns the lowercased domain of an address that is the
// right shape. The shape test is the same one the page makes, on purpose: the
// only difference between the two checks is the domain, and keeping them
// otherwise identical is what makes the late refusal a lesson about where
// validation lives rather than a second, unrelated rule.
func wizardEmailDomain(email string) (string, bool) {
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || strings.ContainsAny(email, " \t") {
		return "", false
	}

	domain := strings.ToLower(email[at+1:])
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", false
	}
	return domain, true
}

func wizardDigits(value string) int {
	count := 0
	for _, r := range value {
		if r >= '0' && r <= '9' {
			count++
		}
	}
	return count
}

func handleWizardDraft(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, wizardFor(session.MustFromContext(r.Context())).view())
}

func handleWizardRecordStep(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Step   int               `json:"step"`
		Values map[string]string `json:"values"`
	}
	decodeJSON(r, &body)

	draft := wizardFor(session.MustFromContext(r.Context()))
	draft.record(body.Step, body.Values)
	httpx.JSON(w, http.StatusOK, draft.view())
}

// handleWizardSubmit reads nothing from the request. A wizard that trusts the
// application the last page hands it is a wizard whose validation can be
// skipped by never walking the steps, so the only draft that can be submitted
// here is the one the server watched being filled in.
func handleWizardSubmit(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())

	app, problems, had := wizardFor(sess).submit(sess.Clock.Now())
	switch {
	case !had:
		httpx.Fail(w, http.StatusConflict,
			"there is no draft to submit; an accepted application spends the draft it was made from")
	case len(problems) > 0:
		httpx.JSON(w, http.StatusUnprocessableEntity, map[string]any{
			"status":   http.StatusUnprocessableEntity,
			"error":    "the application does not validate",
			"problems": problems,
		})
	default:
		httpx.JSON(w, http.StatusCreated, app)
	}
}
