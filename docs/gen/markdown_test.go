package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// noLinks fails any document that tries to link, so a case that is not about
// links cannot pass by accident because its target happened to resolve.
func noLinks(target string) (string, error) {
	return "", errUnexpected(target)
}

type errUnexpected string

func (e errUnexpected) Error() string { return "unexpected link to " + string(e) }

func noHoles(name string) (string, []heading, error) {
	return "", nil, errUnexpected(name)
}

func mustRender(t *testing.T, source string) string {
	t.Helper()
	body, _, err := render(source, noLinks, noHoles)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	return body
}

func TestRenderBlocks(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   string
	}{
		{
			"paragraph joins wrapped lines",
			"one two\nthree four",
			"<p>one two\nthree four</p>\n",
		},
		{
			"tight list stays tight",
			"- first\n- second",
			"<ul>\n<li>first</li>\n<li>second</li>\n</ul>\n",
		},
		{
			"a wrapped list item keeps its second line",
			"- first line\n  second line\n- next",
			"<ul>\n<li>first line\nsecond line</li>\n<li>next</li>\n</ul>\n",
		},
		{
			"a numbered list that does not start at one says so",
			"3. third\n4. fourth",
			"<ol start=\"3\">\n<li>third</li>\n<li>fourth</li>\n</ol>\n",
		},
		{
			"a nested list is a child of its item",
			"- outer\n  - inner",
			"<ul>\n<li><p>outer</p>\n<ul>\n<li>inner</li>\n</ul></li>\n</ul>\n",
		},
		{
			"a fenced block inside an item keeps its indentation",
			"- run this:\n\n  ```sh\n  make\n  ```",
			"<ul>\n<li><p>run this:</p>\n<pre><code class=\"language-sh\">make\n</code></pre></li>\n</ul>\n",
		},
		{
			"a fence escapes what it contains",
			"```go\nif a < b && c {\n```",
			"<pre><code class=\"language-go\">if a &lt; b &amp;&amp; c {\n</code></pre>\n",
		},
		{
			"a table needs its divider",
			"| a | b |\n|---|---|\n| 1 | 2 |",
			"<div class=\"table-scroll\">\n<table>\n<thead>\n<tr><th>a</th><th>b</th></tr>\n" +
				"</thead>\n<tbody>\n<tr><td>1</td><td>2</td></tr>\n</tbody>\n</table>\n</div>\n",
		},
		{
			"a table divider carries alignment",
			"| a | b |\n|---|--:|\n| 1 | 2 |",
			"<div class=\"table-scroll\">\n<table>\n<thead>\n" +
				"<tr><th>a</th><th style=\"text-align:right\">b</th></tr>\n" +
				"</thead>\n<tbody>\n<tr><td>1</td><td style=\"text-align:right\">2</td></tr>\n" +
				"</tbody>\n</table>\n</div>\n",
		},
		{
			"a line of pipes without a divider is prose",
			"a | b is not a table",
			"<p>a | b is not a table</p>\n",
		},
		{
			"a list directly after a paragraph is not swallowed by it",
			"lead in:\n- one",
			"<p>lead in:</p>\n<ul>\n<li>one</li>\n</ul>\n",
		},
		{
			"a blockquote renders its own blocks",
			"> quoted",
			"<blockquote>\n<p>quoted</p>\n</blockquote>\n",
		},
		{
			"a rule is a rule",
			"---",
			"<hr>\n",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mustRender(t, c.source); got != c.want {
				t.Errorf("\n got: %q\nwant: %q", got, c.want)
			}
		})
	}
}

func TestRenderInline(t *testing.T) {
	cases := []struct{ name, source, want string }{
		{"code span", "a `b` c", "<p>a <code>b</code> c</p>\n"},
		{"code span may contain a backtick", "a `` ` `` c", "<p>a <code>`</code> c</p>\n"},
		{"code span is escaped, not parsed", "`<b>*x*</b>`",
			"<p><code>&lt;b&gt;*x*&lt;/b&gt;</code></p>\n"},
		{"strong", "**loud**", "<p><strong>loud</strong></p>\n"},
		{"emphasis", "*quiet*", "<p><em>quiet</em></p>\n"},
		// An asterisk standing for a route pattern is far more common in this
		// documentation than an italic, and turning the rest of a paragraph
		// italic because of one would be silent.
		{"a lone asterisk is literal", "match /api/* only", "<p>match /api/* only</p>\n"},
		{"an asterisk followed by a space opens nothing", "2 * 3 * 4", "<p>2 * 3 * 4</p>\n"},
		// Identifiers in this documentation are full of underscores, and half an
		// italic identifier is worse than no italics at all.
		{"underscores are literal", "coverage_test.go and _x_",
			"<p>coverage_test.go and _x_</p>\n"},
		{"angle brackets are escaped", "a <b> & c", "<p>a &lt;b&gt; &amp; c</p>\n"},
		{"a backslash escapes the next character", `\*not italic\*`,
			"<p>*not italic*</p>\n"},
		{"raw html is shown rather than passed through", "<script>x</script>",
			"<p>&lt;script&gt;x&lt;/script&gt;</p>\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mustRender(t, c.source); got != c.want {
				t.Errorf("\n got: %q\nwant: %q", got, c.want)
			}
		})
	}
}

func TestRenderHeadings(t *testing.T) {
	body, contents, err := render("# Title\n\n## Endpoints\n\n### Sub\n\n## Endpoints\n",
		noLinks, noHoles)
	if err != nil {
		t.Fatal(err)
	}

	// A duplicated heading that reused the same id would leave one of the two
	// unreachable by the link the sidebar publishes for it.
	for _, want := range []string{`id="title"`, `id="endpoints"`, `id="sub"`, `id="endpoints-2"`} {
		if !strings.Contains(body, want) {
			t.Errorf("body is missing %s\n%s", want, body)
		}
	}

	// The level one heading is the page title and is not a section of itself.
	wantContents := []string{"endpoints", "sub", "endpoints-2"}
	if len(contents) != len(wantContents) {
		t.Fatalf("got %d contents entries, want %d: %+v", len(contents), len(wantContents), contents)
	}
	for i, want := range wantContents {
		if contents[i].ID != want {
			t.Errorf("contents[%d] is %q, want %q", i, contents[i].ID, want)
		}
	}
}

func TestRenderReportsFailures(t *testing.T) {
	t.Run("an unterminated fence", func(t *testing.T) {
		if _, _, err := render("```sh\nmake\n", noLinks, noHoles); err == nil {
			t.Fatal("want an error")
		}
	})

	t.Run("a link the resolver rejects", func(t *testing.T) {
		if _, _, err := render("[gone](nowhere.md)", noLinks, noHoles); err == nil {
			t.Fatal("want an error")
		}
	})

	t.Run("a hole the generator does not know", func(t *testing.T) {
		if _, _, err := render("<!--generated:nothing-->", noLinks, noHoles); err == nil {
			t.Fatal("want an error")
		}
	})
}

// TestEveryPageSourceExists guards the one thing the page table cannot check
// for itself. A renamed or moved source is otherwise found by whoever next
// builds the site, which on a documentation repository is whoever next
// deploys it.
func TestEveryPageSourceExists(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	slugs := map[string]bool{}
	for _, p := range pages {
		if slugs[p.Slug] {
			t.Errorf("two pages claim the slug %q, so one would overwrite the other", p.Slug)
		}
		slugs[p.Slug] = true

		if p.Nav == "" || p.Lead == "" {
			t.Errorf("page %q has no sidebar label or no lead", p.Slug)
		}
		source := filepath.Join(root, filepath.FromSlash(p.Source))
		if _, err := os.Stat(source); err != nil {
			t.Errorf("page %q: %v", p.Slug, err)
		}
	}
}
