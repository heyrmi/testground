package main

import (
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"
)

// This renders a deliberately small Markdown subset rather than depending on a
// parser. A Markdown library would land in the go.mod of a tool people audit
// before running it inside a corporate network, in exchange for a build step
// that never reaches the binary, and the project caps its dependencies at ten
// for exactly that reason.
//
// The cost of the choice is that a document can reach for a construct this
// does not understand. The mitigation is that nothing is dropped silently:
// anything unrecognised falls through to a paragraph and is escaped, so a page
// that strays outside the subset reads as obviously wrong rather than quietly
// losing a sentence. What is supported is written down in docs/site.md, which
// is the page a contributor lands on before adding one.

// heading is one entry in a page's table of contents. Text is already rendered
// and escaped, because a section named after a `data-testid` should read the
// same in the sidebar as it does in the page.
type heading struct {
	Level int
	Text  template.HTML
	ID    string
}

// linkFunc turns a Markdown link target into a URL in the built site. It
// returns an error rather than a best guess for a target it cannot place,
// because internal links that quietly point at nothing are the rot this site
// exists to avoid.
type linkFunc func(target string) (string, error)

// holeFunc renders one <!--generated:name--> placeholder, returning its HTML
// and any headings it introduced. The placeholder is an HTML comment so that a
// source file still reads correctly on GitHub, where the generated section
// simply is not there.
//
// A generated section reports its headings so that the contents sidebar covers
// the whole page. A sidebar that silently stops at the last hand-written
// heading would be at its least useful on the longest page in the site.
type holeFunc func(name string) (string, []heading, error)

var (
	headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*$`)
	itemPattern    = regexp.MustCompile(`^(\s*)([-*+]|\d{1,9}[.)])(\s+)(.*)$`)
	rulePattern    = regexp.MustCompile(`^ {0,3}(?:-{3,}|\*{3,}|_{3,})\s*$`)
	holePattern    = regexp.MustCompile(`^<!--\s*generated:\s*([a-z0-9-]+)\s*-->\s*$`)
	dividerPattern = regexp.MustCompile(`^:?-+:?$`)
	slugPattern    = regexp.MustCompile(`[^a-z0-9]+`)
)

// renderer carries the state one document needs. Heading ids are deduplicated
// per document rather than globally, so two pages may both have an "Endpoints"
// section without one of them getting an id nobody would guess.
type renderer struct {
	link linkFunc
	hole holeFunc
	ids  map[string]int
	toc  []heading
	err  error
}

// render turns one Markdown document into the HTML body of a page, along with
// the headings worth listing in a contents sidebar.
func render(source string, link linkFunc, hole holeFunc) (string, []heading, error) {
	r := &renderer{link: link, hole: hole, ids: map[string]int{}}
	var out strings.Builder
	r.blocks(splitLines(source), &out)
	if r.err != nil {
		return "", nil, r.err
	}
	return out.String(), r.toc, nil
}

// splitLines normalises the line endings a document may arrive with, so a file
// checked out on Windows does not render every paragraph with a stray carriage
// return inside it.
func splitLines(source string) []string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\r", "\n")
	return strings.Split(source, "\n")
}

// fail records the first problem and lets the rest of the pass run. Reporting
// the first one is enough to stop the build, and continuing avoids a second
// error that is only a consequence of the first.
func (r *renderer) fail(format string, args ...any) {
	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
}

func (r *renderer) blocks(lines []string, out *strings.Builder) {
	for i := 0; i < len(lines); {
		line := lines[i]
		switch {
		case strings.TrimSpace(line) == "":
			i++
		case isFence(line):
			i = r.fence(lines, i, out)
		case holePattern.MatchString(strings.TrimSpace(line)):
			i = r.generated(lines, i, out)
		case headingPattern.MatchString(line):
			i = r.heading(lines, i, out)
		case rulePattern.MatchString(line):
			out.WriteString("<hr>\n")
			i++
		case strings.HasPrefix(strings.TrimSpace(line), ">"):
			i = r.quote(lines, i, out)
		case isTable(lines, i):
			i = r.table(lines, i, out)
		case itemPattern.MatchString(line):
			i = r.list(lines, i, out)
		default:
			i = r.paragraph(lines, i, out)
		}
	}
}

// startsBlock reports whether a line would begin something other than the
// paragraph currently being collected. Paragraphs otherwise run to the next
// blank line, which would swallow a table or a list that follows one directly.
func startsBlock(lines []string, i int) bool {
	line := lines[i]
	switch {
	case strings.TrimSpace(line) == "",
		isFence(line),
		holePattern.MatchString(strings.TrimSpace(line)),
		headingPattern.MatchString(line),
		rulePattern.MatchString(line),
		strings.HasPrefix(strings.TrimSpace(line), ">"),
		isTable(lines, i),
		itemPattern.MatchString(line):
		return true
	}
	return false
}

func isFence(line string) bool {
	trimmed := strings.TrimLeft(line, " ")
	return strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~")
}

func (r *renderer) fence(lines []string, i int, out *strings.Builder) int {
	trimmed := strings.TrimLeft(lines[i], " ")
	marker := trimmed[:1]
	width := len(trimmed) - len(strings.TrimLeft(trimmed, marker))
	language := strings.TrimSpace(trimmed[width:])
	indent := len(lines[i]) - len(trimmed)

	var body []string
	closed := false
	j := i + 1
	for ; j < len(lines); j++ {
		candidate := strings.TrimSpace(lines[j])
		if strings.HasPrefix(candidate, strings.Repeat(marker, width)) &&
			strings.Trim(candidate, marker) == "" {
			closed, j = true, j+1
			break
		}
		body = append(body, trimIndent(lines[j], indent))
	}
	if !closed {
		r.fail("unterminated %s code fence", marker)
	}

	out.WriteString("<pre><code")
	if language != "" {
		fmt.Fprintf(out, " class=%q", "language-"+language)
	}
	out.WriteString(">")
	out.WriteString(escape(strings.Join(body, "\n")))
	if len(body) > 0 {
		out.WriteString("\n")
	}
	out.WriteString("</code></pre>\n")
	return j
}

// trimIndent removes up to n leading spaces, which is what lets a fenced block
// inside a list item keep its own relative indentation.
func trimIndent(line string, n int) string {
	for ; n > 0 && strings.HasPrefix(line, " "); n-- {
		line = line[1:]
	}
	return line
}

func (r *renderer) generated(lines []string, i int, out *strings.Builder) int {
	name := holePattern.FindStringSubmatch(strings.TrimSpace(lines[i]))[1]
	html, contents, err := r.hole(name)
	if err != nil {
		r.fail("generated section %q: %w", name, err)
		return i + 1
	}
	out.WriteString(html)
	r.toc = append(r.toc, contents...)
	return i + 1
}

func (r *renderer) heading(lines []string, i int, out *strings.Builder) int {
	match := headingPattern.FindStringSubmatch(lines[i])
	level := len(match[1])
	text := r.inline(match[2])
	id := r.slug(match[2])

	fmt.Fprintf(out, "<h%d id=%q>%s", level, id, text)
	// Only the levels the contents sidebar lists get a self link. A link on a
	// heading nobody can navigate to from the sidebar is decoration.
	if level == 2 || level == 3 {
		fmt.Fprintf(out, `<a class="anchor" href="#%s" aria-label="Link to this section">#</a>`, id)
		r.toc = append(r.toc, heading{Level: level, Text: template.HTML(text), ID: id})
	}
	fmt.Fprintf(out, "</h%d>\n", level)
	return i + 1
}

// slug derives a heading's fragment from its text so that a link written today
// keeps working, and disambiguates repeats rather than emitting a duplicate id
// that only one of the two headings would ever be reachable by.
func (r *renderer) slug(text string) string {
	base := slugPattern.ReplaceAllString(strings.ToLower(stripInline(text)), "-")
	base = strings.Trim(base, "-")
	if base == "" {
		base = "section"
	}
	r.ids[base]++
	if n := r.ids[base]; n > 1 {
		return base + "-" + strconv.Itoa(n)
	}
	return base
}

// stripInline removes the inline markers before slugging, so a heading that
// emphasises a word does not carry the asterisks into its fragment.
func stripInline(text string) string {
	return strings.NewReplacer("`", "", "*", "", "[", "", "]", "").Replace(text)
}

func (r *renderer) quote(lines []string, i int, out *strings.Builder) int {
	var body []string
	j := i
	for ; j < len(lines); j++ {
		trimmed := strings.TrimSpace(lines[j])
		if !strings.HasPrefix(trimmed, ">") {
			break
		}
		body = append(body, strings.TrimPrefix(strings.TrimPrefix(trimmed, ">"), " "))
	}
	out.WriteString("<blockquote>\n")
	r.blocks(body, out)
	out.WriteString("</blockquote>\n")
	return j
}

// isTable requires a header row and a divider row together. A single line
// containing pipes is far more likely to be prose about the route pattern
// language than the start of a table.
func isTable(lines []string, i int) bool {
	if i+1 >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
		return false
	}
	header, divider := splitRow(lines[i]), splitRow(lines[i+1])
	if len(divider) == 0 || len(header) != len(divider) {
		return false
	}
	for _, cell := range divider {
		if !dividerPattern.MatchString(cell) {
			return false
		}
	}
	return true
}

// splitRow cuts a table row into cells, honouring a backslash-escaped pipe so
// that a cell can contain the route pattern separator without ending itself.
func splitRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimSuffix(strings.TrimPrefix(trimmed, "|"), "|")

	var (
		cells   []string
		current strings.Builder
	)
	for i := 0; i < len(trimmed); i++ {
		switch {
		case trimmed[i] == '\\' && i+1 < len(trimmed) && trimmed[i+1] == '|':
			current.WriteByte('|')
			i++
		case trimmed[i] == '|':
			cells = append(cells, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteByte(trimmed[i])
		}
	}
	return append(cells, strings.TrimSpace(current.String()))
}

func (r *renderer) table(lines []string, i int, out *strings.Builder) int {
	header := splitRow(lines[i])
	aligns := alignments(splitRow(lines[i+1]))

	out.WriteString("<div class=\"table-scroll\">\n<table>\n<thead>\n<tr>")
	for column, cell := range header {
		fmt.Fprintf(out, "<th%s>%s</th>", aligns[column], r.inline(cell))
	}
	out.WriteString("</tr>\n</thead>\n<tbody>\n")

	j := i + 2
	for ; j < len(lines); j++ {
		if !strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
			break
		}
		out.WriteString("<tr>")
		for column, cell := range splitRow(lines[j]) {
			if column >= len(header) {
				break
			}
			fmt.Fprintf(out, "<td%s>%s</td>", aligns[column], r.inline(cell))
		}
		out.WriteString("</tr>\n")
	}
	out.WriteString("</tbody>\n</table>\n</div>\n")
	return j
}

func alignments(divider []string) []string {
	out := make([]string, len(divider))
	for i, cell := range divider {
		left, right := strings.HasPrefix(cell, ":"), strings.HasSuffix(cell, ":")
		switch {
		case left && right:
			out[i] = ` style="text-align:center"`
		case right:
			out[i] = ` style="text-align:right"`
		}
	}
	return out
}

func (r *renderer) list(lines []string, i int, out *strings.Builder) int {
	first := itemPattern.FindStringSubmatch(lines[i])
	indent := len(first[1])
	ordered := first[2][0] >= '0' && first[2][0] <= '9'

	tag := "ul"
	open := "<ul>\n"
	if ordered {
		tag = "ol"
		open = "<ol>\n"
		if start := strings.TrimRight(first[2], ".)"); start != "1" {
			open = fmt.Sprintf("<ol start=%q>\n", start)
		}
	}

	items := [][]string{{first[4]}}
	content := len(first[1]) + len(first[2]) + len(first[3])
	j := i + 1
collect:
	for ; j < len(lines); j++ {
		line := lines[j]
		match := itemPattern.FindStringSubmatch(line)
		switch {
		case match != nil && len(match[1]) == indent:
			items = append(items, []string{match[4]})
			content = len(match[1]) + len(match[2]) + len(match[3])
		case strings.TrimSpace(line) == "":
			items[len(items)-1] = append(items[len(items)-1], "")
		case leadingSpaces(line) >= content:
			items[len(items)-1] = append(items[len(items)-1], line[content:])
		// A line that is neither indented nor a new marker still belongs to the
		// item as long as the item has not been interrupted by a blank line.
		// Prose wrapped at the margin is the common case here, and losing the
		// second half of a sentence to a stray paragraph would be silent.
		case strings.TrimSpace(lines[j-1]) != "":
			items[len(items)-1] = append(items[len(items)-1], strings.TrimSpace(line))
		default:
			break collect
		}
	}

	out.WriteString(open)
	for _, item := range items {
		var body strings.Builder
		r.blocks(trimTrailingBlanks(item), &body)
		out.WriteString("<li>" + unwrapParagraph(body.String()) + "</li>\n")
	}
	fmt.Fprintf(out, "</%s>\n", tag)
	return j
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func trimTrailingBlanks(lines []string) []string {
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// unwrapParagraph keeps a list of one-paragraph items tight. Leaving the <p> in
// place would space every bullet as if it were its own section, which is the
// wrong reading for the short lists this documentation is made of.
func unwrapParagraph(html string) string {
	body := strings.TrimSpace(html)
	if !strings.HasPrefix(body, "<p>") || !strings.HasSuffix(body, "</p>") {
		return body
	}
	inner := body[len("<p>") : len(body)-len("</p>")]
	if strings.Contains(inner, "<p>") {
		return body
	}
	return inner
}

func (r *renderer) paragraph(lines []string, i int, out *strings.Builder) int {
	var body []string
	j := i
	for ; j < len(lines); j++ {
		if j > i && startsBlock(lines, j) {
			break
		}
		body = append(body, strings.TrimSpace(lines[j]))
	}
	out.WriteString("<p>" + r.inline(strings.Join(body, "\n")) + "</p>\n")
	return j
}

// inline renders the four inline forms the subset allows: code spans, strong,
// emphasis and links. Underscores are deliberately not emphasis, because file
// and field names in this documentation are full of them and a rule that turns
// half an identifier italic is worse than no italics at all.
func (r *renderer) inline(text string) string {
	var out strings.Builder
	for i := 0; i < len(text); {
		switch text[i] {
		case '\\':
			if i+1 < len(text) {
				out.WriteString(escape(text[i+1 : i+2]))
				i += 2
				continue
			}
			out.WriteString(`\`)
			i++
		case '`':
			if width, body, next := codeSpan(text, i); width > 0 {
				out.WriteString("<code>" + escape(body) + "</code>")
				i = next
				continue
			}
			out.WriteString("&#96;")
			i++
		case '*':
			if body, next, strong := emphasis(text, i); next > i {
				tag := "em"
				if strong {
					tag = "strong"
				}
				fmt.Fprintf(&out, "<%s>%s</%s>", tag, r.inline(body), tag)
				i = next
				continue
			}
			out.WriteString("*")
			i++
		case '[':
			if label, target, next := linkParts(text, i); next > i {
				href, err := r.link(target)
				if err != nil {
					r.fail("link %q: %w", target, err)
					return out.String()
				}
				fmt.Fprintf(&out, "<a href=%q%s>%s</a>",
					href, externalAttributes(href), r.inline(label))
				i = next
				continue
			}
			out.WriteString("[")
			i++
		default:
			out.WriteString(escape(text[i : i+1]))
			i++
		}
	}
	return out.String()
}

// externalAttributes marks the links that leave the site. Anything absolute
// leaves it, since the whole site is relative by construction so that it opens
// from a file:// path as readily as from Pages.
func externalAttributes(href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return ` rel="noopener noreferrer"`
	}
	return ""
}

// codeSpan matches a run of backticks with an equal run, so that a span can
// itself contain a backtick.
func codeSpan(text string, i int) (width int, body string, next int) {
	width = len(text[i:]) - len(strings.TrimLeft(text[i:], "`"))
	closing := strings.Repeat("`", width)
	rest := text[i+width:]
	for at := 0; ; {
		found := strings.Index(rest[at:], closing)
		if found < 0 {
			return 0, "", i
		}
		found += at
		if found+width < len(rest) && rest[found+width] == '`' {
			at = found + width
			for at < len(rest) && rest[at] == '`' {
				at++
			}
			continue
		}
		body = rest[:found]
		if strings.HasPrefix(body, " ") && strings.HasSuffix(body, " ") &&
			strings.TrimSpace(body) != "" {
			body = body[1 : len(body)-1]
		}
		return width, body, i + width + found + width
	}
}

// emphasis reads *this* or **this**. The closing marker must not be preceded by
// a space, which is what keeps a lone asterisk in prose from opening a span
// that runs to the end of the paragraph.
func emphasis(text string, i int) (body string, next int, strong bool) {
	width := 1
	if strings.HasPrefix(text[i:], "**") {
		width, strong = 2, true
	}
	marker := strings.Repeat("*", width)
	rest := text[i+width:]
	if rest == "" || rest[0] == ' ' {
		return "", i, false
	}
	found := strings.Index(rest, marker)
	if found <= 0 || rest[found-1] == ' ' {
		return "", i, false
	}
	return rest[:found], i + width + found + width, strong
}

// linkParts reads [label](target), counting brackets so a label may contain a
// bracketed aside without ending the link early.
func linkParts(text string, i int) (label, target string, next int) {
	depth := 0
	for at := i; at < len(text); at++ {
		switch text[at] {
		case '[':
			depth++
		case ']':
			depth--
			if depth > 0 {
				continue
			}
			if at+1 >= len(text) || text[at+1] != '(' {
				return "", "", i
			}
			end := strings.IndexByte(text[at+2:], ')')
			if end < 0 {
				return "", "", i
			}
			return text[i+1 : at], strings.TrimSpace(text[at+2 : at+2+end]), at + 2 + end + 1
		}
	}
	return "", "", i
}

var escaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	`"`, "&quot;",
)

func escape(text string) string { return escaper.Replace(text) }
