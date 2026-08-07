// Command gen builds the testground documentation site.
//
// It exists instead of MkDocs or Docusaurus because this project ships as one
// Go binary with everything vendored and no CDN anywhere, and a documentation
// site that needed a Python or Node toolchain to build, or fetched a theme at
// view time, would be the first place that promise stopped being true. What
// the generator needs is already installed by anyone who can compile the
// playground, it adds nothing to go.mod, and the output is static HTML with
// one hand-written stylesheet and no script at all, so the site opens from a
// file:// path with no server and no network.
//
// The reasoning is written up for readers rather than maintainers in
// docs/site.md, which is a page of the site.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/heyrmi/testground/internal/challenge"
)

//go:embed assets
var assets embed.FS

//go:embed layout.html
var layoutSource string

// repoURL is where a link that does not name a page of this site is sent. The
// documentation refers to files that are worth reading and are not worth
// republishing, and pointing at the repository is better than either omitting
// the reference or copying the file and letting the copy rot.
const repoURL = "https://github.com/heyrmi/testground/blob/main/"

// page is one document in the site. The set is declared here as data, in the
// order it is presented, so that adding a page is one line and a page nobody
// listed cannot end up published with no way to reach it.
//
// Source is relative to the repository root rather than to docs/, which is what
// lets CONTRIBUTING.md be a page of this site instead of being copied into it.
// A copy is what went stale last time.
type page struct {
	Slug   string
	Source string
	Nav    string
	Lead   string
}

var pages = []page{
	{"index", "docs/index.md", "Overview",
		"What the playground is, and the four promises it makes"},
	{"getting-started", "docs/getting-started.md", "Getting started",
		"Install it, run it, and drive it from a suite"},
	{"zones", "docs/zones.md", "The zone model",
		"Why one server hosts several frontends at once"},
	{"catalogue", "docs/catalogue.md", "Challenge catalogue",
		"Every challenge, generated from the manifest"},
	{"control-plane", "docs/control-plane.md", "The control plane",
		"Latency, failure, flake and the clock, per session"},
	{"stability-contract", "docs/stability-contract.md", "The stability contract",
		"What a released challenge promises never to change"},
	{"contributing", "CONTRIBUTING.md", "Contributing",
		"Adding a challenge, and the six rules it must follow"},
	{"site", "docs/site.md", "About this site",
		"How the documentation is built and published"},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "docs: "+err.Error())
		os.Exit(1)
	}
}

func run() error {
	var (
		out      string
		manifest string
		addr     string
	)
	flag.StringVar(&out, "out", "", "directory to write the site into (default docs/_site)")
	flag.StringVar(&manifest, "manifest", "", "manifest JSON to build the catalogue from (default: run the playground manifest command)")
	flag.StringVar(&addr, "serve", "", "serve the built site on this address, for example 127.0.0.1:7575")
	flag.Parse()

	root, err := repositoryRoot()
	if err != nil {
		return err
	}
	if out == "" {
		out = filepath.Join(root, "docs", "_site")
	}

	catalogue, err := loadManifest(root, manifest)
	if err != nil {
		return err
	}

	site := &site{root: root, out: out, manifest: catalogue}
	if err := site.build(); err != nil {
		return err
	}
	fmt.Printf("docs: wrote %d pages and %d challenges to %s\n",
		len(pages), catalogue.Count, out)

	if addr == "" {
		return nil
	}
	fmt.Printf("docs: serving %s on http://%s/\n", out, addr)
	return http.ListenAndServe(addr, http.FileServer(http.Dir(out)))
}

// repositoryRoot walks up from the working directory looking for go.mod, so the
// generator can be run from anywhere rather than only from the one directory
// its relative paths happen to be written against.
func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no go.mod above the working directory; run this from inside the repository")
		}
		dir = parent
	}
}

// loadManifest reads the catalogue from a file, or asks the playground for it.
//
// Running the command is the default because a catalogue built from anything
// other than the manifest is a second, hand-maintained list of the same facts,
// and the last one of those drifted. The file form is for a build that already
// has a compiled binary and would rather not compile a second one.
func loadManifest(root, file string) (challenge.Manifest, error) {
	var body []byte
	var err error

	if file != "" {
		body, err = os.ReadFile(file)
		if err != nil {
			return challenge.Manifest{}, err
		}
	} else {
		command := exec.Command("go", "run", "./cmd/playground", "manifest")
		command.Dir = root
		var stderr bytes.Buffer
		command.Stderr = &stderr
		body, err = command.Output()
		if err != nil {
			return challenge.Manifest{}, fmt.Errorf("running the manifest command: %w\n%s", err, stderr.String())
		}
	}

	var manifest challenge.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return challenge.Manifest{}, fmt.Errorf("parsing the manifest: %w", err)
	}
	if manifest.Count == 0 {
		return challenge.Manifest{}, errors.New("the manifest describes no challenges")
	}
	return manifest, nil
}

type site struct {
	root     string
	out      string
	manifest challenge.Manifest
}

type navLink struct {
	Label   string
	Lead    string
	Href    string
	Current bool
}

type view struct {
	Title    string
	Lead     string
	Nav      []navLink
	Contents []heading
	Body     template.HTML
	Version  string
}

func (s *site) build() error {
	if err := os.RemoveAll(s.out); err != nil {
		return err
	}
	if err := os.MkdirAll(s.out, 0o755); err != nil {
		return err
	}
	layout, err := template.New("layout").Parse(layoutSource)
	if err != nil {
		return err
	}
	for _, p := range pages {
		if err := s.page(layout, p); err != nil {
			return fmt.Errorf("%s: %w", p.Source, err)
		}
	}
	return s.copyAssets()
}

func (s *site) page(layout *template.Template, p page) error {
	source, err := os.ReadFile(filepath.Join(s.root, filepath.FromSlash(p.Source)))
	if err != nil {
		return err
	}

	body, contents, err := render(string(source),
		func(target string) (string, error) { return s.resolve(p, target) },
		s.hole)
	if err != nil {
		return err
	}

	title, err := documentTitle(string(source))
	if err != nil {
		return err
	}

	var rendered bytes.Buffer
	err = layout.Execute(&rendered, view{
		Title:    title,
		Lead:     p.Lead,
		Nav:      s.navigation(p),
		Contents: contents,
		Body:     template.HTML(body),
		Version:  s.versionLabel(),
	})
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.out, p.Slug+".html"), rendered.Bytes(), 0o644)
}

// documentTitle takes the page title from its first heading rather than from
// the declaration above, so that the name in the browser tab and the name at
// the top of the page cannot disagree.
func documentTitle(source string) (string, error) {
	for _, line := range splitLines(source) {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(stripInline(line[2:])), nil
		}
	}
	return "", errors.New("no level one heading, so the page has no title")
}

func (s *site) navigation(current page) []navLink {
	links := make([]navLink, 0, len(pages))
	for _, p := range pages {
		links = append(links, navLink{
			Label:   p.Nav,
			Lead:    p.Lead,
			Href:    p.Slug + ".html",
			Current: p.Slug == current.Slug,
		})
	}
	return links
}

// resolve turns a Markdown link target into a URL in the built site.
//
// A target naming a page becomes a relative link, which is what lets the site
// be read from a file:// path and from a Pages subdirectory without either of
// them needing to know its own base URL. Anything else must at least exist in
// the repository: a link that resolves to nothing is the failure this site is
// meant to be an antidote to, so it stops the build rather than shipping.
func (s *site) resolve(from page, target string) (string, error) {
	if target == "" {
		return "", errors.New("empty target")
	}
	if strings.HasPrefix(target, "#") ||
		strings.HasPrefix(target, "http://") ||
		strings.HasPrefix(target, "https://") ||
		strings.HasPrefix(target, "mailto:") {
		return target, nil
	}

	file, fragment, _ := strings.Cut(target, "#")
	if fragment != "" {
		fragment = "#" + fragment
	}
	resolved := path.Clean(path.Join(path.Dir(from.Source), file))

	for _, p := range pages {
		if p.Source == resolved {
			return p.Slug + ".html" + fragment, nil
		}
	}
	if _, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(resolved))); err != nil {
		return "", fmt.Errorf("%q names neither a page of this site nor a file in the repository", target)
	}
	return repoURL + resolved + fragment, nil
}

func (s *site) copyAssets() error {
	return fs.WalkDir(assets, "assets", func(name string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		body, err := assets.ReadFile(name)
		if err != nil {
			return err
		}
		target := filepath.Join(s.out, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, body, 0o644)
	})
}

// sortedUnique is used wherever a count is derived from the manifest, because
// a figure the documentation quotes has to be the same on every machine that
// builds it or a rebuild becomes an unreviewable diff.
func sortedUnique(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}
