package classic

import (
	"net/http"
	"strings"

	"github.com/heyrmi/testground/internal/challenge"
)

type choiceValues struct {
	Toppings   string
	Delivery   string
	Country    string
	Languages  string
	Newsletter string
}

func choices() page {
	meta := challenge.Challenge{
		ID:       "choices",
		Title:    "Checkboxes, radios and selects",
		URL:      "/classic/choices",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T1,
		Category: "A. Basic Controls",
		Summary: "A checkbox group sharing one name, a radio group, a select with option " +
			"groups and a disabled option, and a multi-select that accepts several values " +
			"at once. One of the checkboxes starts checked.",
		WhyHard: "A multi-select is not a click target: choosing more than one option needs " +
			"the framework's select API rather than clicks, and the modifier key that does " +
			"it by hand differs per platform. The disabled option looks selectable and is " +
			"not. The checkbox group shares a single name across several inputs, so asking " +
			"for the group's value returns whichever box the locator found first rather " +
			"than the set that is actually checked.",
		Hint: "Select options through the select API and assert on the resulting value list. " +
			"For the checkbox group, assert on which boxes are checked rather than on one " +
			"value -- the form posts a repeated field, not a single one. The disabled " +
			"option carries the attribute; read it instead of discovering it by clicking.",
		Tags:     []string{"forms", "inputs", "select", "checkbox", "radio"},
		Concepts: []string{"repeated form fields", "multi-select APIs", "disabled options", "default checked state"},
		Selectors: []challenge.Selector{
			{TestID: "topping-cheese", Role: "checkbox", Note: "Starts checked; shares the name topping with its siblings"},
			{TestID: "topping-olives", Role: "checkbox", Note: "Same name, different value"},
			{TestID: "topping-anchovies", Role: "checkbox", Note: "Same name again"},
			{TestID: "delivery-standard", Role: "radio", Note: "Radio group; selecting one clears the others"},
			{TestID: "delivery-express", Role: "radio", Note: "Second option in the same group"},
			{TestID: "field-country", Role: "combobox", Note: "Single select with optgroup headings and one disabled option"},
			{TestID: "field-languages", Role: "listbox", Note: "Multi-select; needs the select API, not clicks"},
			{TestID: "field-newsletter", Role: "checkbox", Note: "Lone checkbox, unchecked by default"},
			{TestID: "submit", Role: "button", Note: "Posts the form"},
			{TestID: "result", Transient: true, Note: "What the server received, absent until posted"},
			{TestID: "no-submission", Note: "Shown while nothing has been posted"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/choices", Note: "Records the submission, answers 303"},
		},
		Stability: challenge.Stable,
	}

	return formPage(meta, func(r *http.Request) choiceValues {
		list := func(name string) string {
			values := r.PostForm[name]
			if len(values) == 0 {
				return ""
			}
			return strings.Join(values, ", ")
		}
		return choiceValues{
			Toppings:   list("topping"),
			Delivery:   r.PostFormValue("delivery"),
			Country:    r.PostFormValue("country"),
			Languages:  list("languages"),
			Newsletter: r.PostFormValue("newsletter"),
		}
	})
}
