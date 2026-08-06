// Package render turns the embedded Go templates into pages.
//
// Each page is parsed into its own template set alongside the layout and the
// shared partials, so a page can override a block without leaking that
// override into every other page.
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"strings"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/session"
)

// View is what every template receives. The renderer fills in the fields that
// describe the caller rather than the page.
type View struct {
	Title     string
	Challenge *challenge.Challenge
	Data      any

	Version string
	Session string
	Seed    uint64
}

// ZoneView is the data a zone index page renders from.
type ZoneView struct {
	Zone       challenge.ZoneInfo
	Challenges []challenge.Challenge
}

// Renderer holds the parsed template sets. It is read-only after New.
type Renderer struct {
	pages   map[string]*template.Template
	version string
	log     *slog.Logger
}

var funcs = template.FuncMap{
	"lower": func(v any) string { return strings.ToLower(fmt.Sprint(v)) },
}

// New parses every template under fsys, which must be rooted at the templates
// directory. Page names are their path below pages/ without the extension, so
// pages/wc/nested-shadow.html renders as "wc/nested-shadow".
func New(fsys fs.FS, version string, log *slog.Logger) (*Renderer, error) {
	shared := []string{"layout.html"}
	partials, err := fs.Glob(fsys, "partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("globbing partials: %w", err)
	}
	shared = append(shared, partials...)

	pages := make(map[string]*template.Template)
	err = fs.WalkDir(fsys, "pages", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".html") {
			return err
		}
		set, parseErr := template.New(path.Base(p)).Funcs(funcs).ParseFS(fsys, append(shared, p)...)
		if parseErr != nil {
			return fmt.Errorf("parsing %s: %w", p, parseErr)
		}
		name := strings.TrimSuffix(strings.TrimPrefix(p, "pages/"), ".html")
		pages[name] = set
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no page templates found")
	}

	return &Renderer{pages: pages, version: version, log: log}, nil
}

// Page renders the named page with a 200.
func (r *Renderer) Page(w http.ResponseWriter, req *http.Request, name string, view View) {
	r.PageStatus(w, req, http.StatusOK, name, view)
}

// PageStatus renders the named page under a chosen status code. The response
// is buffered so a template that fails halfway cannot emit a broken document
// with a 200 already on the wire.
func (r *Renderer) PageStatus(w http.ResponseWriter, req *http.Request, status int, name string, view View) {
	set, ok := r.pages[name]
	if !ok {
		r.fail(w, req, name, fmt.Errorf("no such page template"))
		return
	}

	view.Version = r.version
	if sess := session.FromContext(req.Context()); sess != nil {
		view.Session = string(sess.ID)
		view.Seed = sess.RNG.Seed()
	}

	var buf bytes.Buffer
	if err := set.ExecuteTemplate(&buf, "layout", view); err != nil {
		r.fail(w, req, name, err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	buf.WriteTo(w)
}

func (r *Renderer) fail(w http.ResponseWriter, req *http.Request, name string, err error) {
	r.log.ErrorContext(req.Context(), "render failed", "page", name, "error", err)
	http.Error(w, "template error: "+name, http.StatusInternalServerError)
}
