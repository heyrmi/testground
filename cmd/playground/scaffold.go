package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/heyrmi/testground/internal/challenge"
)

// idPattern is the same shape the URLs use, because the id becomes one. A
// challenge whose id needs escaping would put an escape into every selector
// note, every spec filename and every manifest lookup.
var idPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// scaffoldFile is one generated file and the template behind it.
type scaffoldFile struct {
	path string
	body *template.Template
}

// scaffoldData is what the templates render from.
type scaffoldData struct {
	ID       string
	Title    string
	Zone     string
	Prefix   string
	Tier     string
	Category string
	URL      string
	Func     string
	// ZoneConst is the Go identifier for the zone rather than its id, which is
	// the one place the two spellings differ: the hx zone is ZoneHypermedia.
	ZoneConst string
	Component string
}

// zoneConsts maps a zone id to the constant that names it, so generated code
// refers to challenge.ZoneComponents rather than to a string the compiler
// cannot check.
var zoneConsts = map[challenge.Zone]string{
	challenge.ZoneClassic:    "Classic",
	challenge.ZoneLegacy:     "Legacy",
	challenge.ZoneHypermedia: "Hypermedia",
	challenge.ZoneApp:        "App",
	challenge.ZoneComponents: "Components",
	challenge.ZoneRealtime:   "Realtime",
}

func newScaffoldCommand() *cobra.Command {
	var (
		zone     string
		tier     string
		category string
		title    string
		root     string
		dryRun   bool
	)

	cmd := &cobra.Command{
		Use:   "scaffold <id>",
		Short: "Generate the files a new challenge needs",
		Long: strings.TrimSpace(`
Generate the files a new challenge needs.

A challenge is three things that must agree: a declaration the registry
validates, a page that renders what the declaration promises, and a reference
test that proves it. Writing them by hand means writing the id four times and
discovering the fourth was wrong when the server refuses to start.

This writes all three with the declaration already filled in, and refuses to
overwrite anything that exists. It deliberately stops short of registering the
challenge, because the two lines that do that are the ones worth reading:
CONTRIBUTING.md explains both, and the command prints them when it finishes.`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if !idPattern.MatchString(id) {
				return fmt.Errorf("id %q must be lowercase words joined by hyphens, since it becomes the URL", id)
			}

			info, ok := challenge.LookupZone(challenge.Zone(zone))
			if !ok {
				return fmt.Errorf("unknown zone %q: use one of %s", zone, strings.Join(zoneIDs(), ", "))
			}
			if !validTier(tier) {
				return fmt.Errorf("unknown tier %q: use T1, T2, T3 or T4", tier)
			}
			if title == "" {
				title = defaultTitle(id)
			}

			data := scaffoldData{
				ID:        id,
				Title:     title,
				Zone:      zone,
				Prefix:    info.Prefix,
				Tier:      tier,
				Category:  category,
				URL:       info.Prefix + "/" + id,
				Func:      lowerCamel(id),
				ZoneConst: zoneConsts[challenge.Zone(zone)],
				Component: upperCamel(id),
			}

			files, err := scaffoldFiles(data)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			for _, file := range files {
				path := filepath.Join(root, file.path)
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("%s already exists; pick another id or remove it first", file.path)
				} else if !errors.Is(err, os.ErrNotExist) {
					return err
				}

				var rendered strings.Builder
				if err := file.body.Execute(&rendered, data); err != nil {
					return err
				}
				if dryRun {
					fmt.Fprintf(out, "\n----- %s -----\n%s", file.path, rendered.String())
					continue
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					return err
				}
				if err := os.WriteFile(path, []byte(rendered.String()), 0o644); err != nil {
					return err
				}
				fmt.Fprintln(out, "wrote", file.path)
			}

			if dryRun {
				return nil
			}
			return writeNextSteps(out, data)
		},
	}

	cmd.Flags().StringVarP(&zone, "zone", "z", "app", "zone to add the challenge to: "+strings.Join(zoneIDs(), ", "))
	cmd.Flags().StringVarP(&tier, "tier", "t", "T2", "difficulty: T1 intro, T2 intermediate, T3 hard, T4 hostile")
	cmd.Flags().StringVarP(&category, "category", "c", "TODO. Category", "catalogue category the challenge belongs to")
	cmd.Flags().StringVar(&title, "title", "", "human title; defaults to the id in sentence case")
	cmd.Flags().StringVar(&root, "dir", ".", "repository root to write into")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print what would be written without writing it")

	return cmd
}

// scaffoldFiles picks the page template by zone. The SPA zone builds its pages
// from React and every other zone from Go templates, so the declaration and the
// spec are shared and only the middle file differs.
func scaffoldFiles(data scaffoldData) ([]scaffoldFile, error) {
	files := []scaffoldFile{
		{
			path: filepath.Join("internal", "zones", data.Zone, snake(data.ID)+".go"),
			body: template.Must(template.New("declaration").Parse(declarationTemplate)),
		},
	}

	if challenge.Zone(data.Zone) == challenge.ZoneApp {
		files = append(files, scaffoldFile{
			path: filepath.Join("web", "app", "src", "app", "challenges", data.ID+".tsx"),
			body: template.Must(template.New("page").Parse(reactPageTemplate)),
		})
	} else {
		files = append(files, scaffoldFile{
			path: filepath.Join("templates", "pages", data.Zone, data.ID+".html"),
			body: template.Must(template.New("page").Parse(goPageTemplate)),
		})
	}

	files = append(files, scaffoldFile{
		path: filepath.Join("examples", "playwright-ts", "tests", data.ID+".spec.ts"),
		body: template.Must(template.New("spec").Parse(specTemplate)),
	})
	return files, nil
}

func writeNextSteps(out interface{ Write([]byte) (int, error) }, data scaffoldData) error {
	steps := []string{
		fmt.Sprintf("add %s() to Challenges() in internal/zones/%s/%s.go", data.Func, data.Zone, data.Zone),
	}
	if challenge.Zone(data.Zone) == challenge.ZoneApp {
		steps = append(steps, "register the route in web/app/src/app/router.tsx, then run make web")
	} else {
		steps = append(steps, fmt.Sprintf("serve the template from internal/zones/%s/%s.go", data.Zone, data.Zone))
	}
	steps = append(steps,
		"replace every TODO: the registry refuses to start while the prose is missing",
		"go test ./... then cd examples/playwright-ts && npm test",
	)

	fmt.Fprintf(out, "\n%s is scaffolded at %s. %d steps left:\n", data.ID, data.URL, len(steps))
	for i, step := range steps {
		fmt.Fprintf(out, "  %d. %s\n", i+1, step)
	}
	return nil
}

func zoneIDs() []string {
	var ids []string
	for _, z := range challenge.Zones() {
		ids = append(ids, string(z.ID))
	}
	return ids
}

func validTier(tier string) bool {
	switch challenge.Tier(tier) {
	case challenge.T1, challenge.T2, challenge.T3, challenge.T4:
		return true
	}
	return false
}

// defaultTitle turns an id into something readable, on the assumption that a
// placeholder nobody edits is better than an empty string the registry rejects
// with a less obvious message.
func defaultTitle(id string) string {
	words := strings.Split(id, "-")
	words[0] = strings.ToUpper(words[0][:1]) + words[0][1:]
	return strings.Join(words, " ")
}

func snake(id string) string { return strings.ReplaceAll(id, "-", "_") }

func lowerCamel(id string) string {
	parts := strings.Split(id, "-")
	for i := 1; i < len(parts); i++ {
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

func upperCamel(id string) string {
	camel := lowerCamel(id)
	return strings.ToUpper(camel[:1]) + camel[1:]
}

const declarationTemplate = `package {{.Zone}}

import "github.com/heyrmi/testground/internal/challenge"

func {{.Func}}() challenge.Challenge {
	return challenge.Challenge{
		ID:       "{{.ID}}",
		Title:    "{{.Title}}",
		URL:      "{{.URL}}",
		Zone:     challenge.Zone{{.ZoneConst}},
		Tier:     challenge.{{.Tier}},
		Category: "{{.Category}}",
		Summary: "TODO. What the page does, in the present tense, for someone who has not " +
			"opened it yet.",
		WhyHard: "TODO. What breaks naive automation here, specifically. Name the failure, " +
			"not the feature.",
		Hint: "TODO. The intended approach as a concept. No framework-specific code: this " +
			"renders on the page and ships in the manifest.",
		Tags:     []string{"TODO"},
		Concepts: []string{"TODO"},
		Selectors: []challenge.Selector{
			{TestID: "TODO", Note: "TODO. Declare every element worth locating; the reference suite looks each one up in the live DOM."},
		},
		Stability: challenge.Experimental,
	}
}
`

const goPageTemplate = `{{"{{"}}define "content" -{{"}}"}}
{{"{{"}}template "challenge-header" .{{"}}"}}
{{"{{"}}template "challenge-panel" .{{"}}"}}

<section class="stage">
  <p class="field__note">
    TODO: build the page. Everything above comes from the manifest, so the
    description and the hint are never written twice.
  </p>
</section>
{{"{{"}}- end{{"}}"}}
`

const reactPageTemplate = `import { createRoute } from '@tanstack/react-router'
import { ChallengePage } from '../chrome'
import { rootRoute } from '../root'

export const route = createRoute({
  getParentRoute: () => rootRoute,
  path: '/{{.ID}}',
  component: {{.Component}},
})

function {{.Component}}() {
  return (
    <ChallengePage id="{{.ID}}">
      {/* TODO: build the page. The chrome, description and hint come from the
          manifest, so they are never written twice. */}
      <p className="text-muted" data-testid="TODO">Not built yet.</p>
    </ChallengePage>
  )
}
`

const specTemplate = `import { expect, test } from './fixtures'

// Show the approach that works. Where the page teaches something, keep one case
// demonstrating the approach that looks like it works and does not.
test('TODO: what this challenge teaches', async ({ page }) => {
  await page.goto('{{.URL}}')

  await expect(page.getByTestId('TODO')).toBeVisible()
})
`
