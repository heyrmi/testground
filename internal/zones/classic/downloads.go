package classic

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"image"
	"image/color"
	"image/png"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/fake"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

const generatedDelayMs = 3000

func downloads() page {
	meta := challenge.Challenge{
		ID:       "downloads",
		Title:    "Downloads, dispositions and a file that takes three seconds to exist",
		URL:      "/classic/downloads",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "N. Files",
		Summary: "A CSV, a ZIP and a PNG generated from your session seed, one file served " +
			"inline instead of as an attachment, one with a non-ASCII filename, and one " +
			"that spends three seconds being generated before a byte is sent.",
		WhyHard: "A download is not a navigation. The click fires a request whose response " +
			"the page never renders, so waiting for the page to change waits forever and " +
			"the test times out on a link that worked perfectly. The filename is in a " +
			"header rather than in the DOM, and once it contains anything outside ASCII it " +
			"is in a second, differently encoded copy of that header, so reading the " +
			"obvious one gives you mojibake. The inline file does not download at all -- " +
			"the browser renders it -- which is a different code path with a different " +
			"failure. And the generated one takes three seconds, so the click finishes " +
			"long before the file does.",
		Hint: "Use the download API your framework provides rather than watching for " +
			"navigation; it hands you the suggested filename and waits for the transfer to " +
			"finish. Read the filename from the response rather than from the link text, " +
			"and expect the non-ASCII one to arrive through filename* rather than " +
			"filename. Content is deterministic from the session seed, so it is safe to " +
			"assert on exactly.",
		Tags:     []string{"files", "download", "content-disposition", "streaming"},
		Concepts: []string{"a download is not a navigation", "Content-Disposition", "RFC 5987 filenames", "inline versus attachment"},
		Selectors: []challenge.Selector{
			{TestID: "download-csv", Role: "link", Note: "Attachment; content is generated from the session seed"},
			{TestID: "download-zip", Role: "link", Note: "Attachment holding two files"},
			{TestID: "download-png", Role: "link", Note: "Attachment; a generated image"},
			{TestID: "download-inline", Role: "link", Note: "Served inline, so the browser renders it instead of saving it"},
			{TestID: "download-unicode", Role: "link", Note: "Filename outside ASCII, which travels in filename* rather than filename"},
			{TestID: "download-slow", Role: "link", Note: "Three seconds of generation before the first byte"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodGet, Path: "/classic/downloads/report.csv", Note: "Deterministic rows from the session seed"},
			{Method: http.MethodGet, Path: "/classic/downloads/bundle.zip", Note: "Two files in an archive"},
			{Method: http.MethodGet, Path: "/classic/downloads/pixel.png", Note: "Generated image"},
			{Method: http.MethodGet, Path: "/classic/downloads/notes.txt", Note: "Content-Disposition: inline"},
			{Method: http.MethodGet, Path: "/classic/downloads/unicode.txt", Note: "Non-ASCII filename via filename*"},
			{Method: http.MethodGet, Path: "/classic/downloads/slow.csv", Note: "Generated after a three-second delay"},
		},
		Controls: []challenge.Control{
			{Name: "rows", Kind: "query", Default: "25", Note: "Rows in the generated CSV, clamped to 1-10000."},
			{Name: "ms", Kind: "query", Default: "3000", Note: "Generation delay on slow.csv, clamped to 0-30000."},
		},
		Stability: challenge.Stable,
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/downloads", render.View{Title: meta.Title, Challenge: &meta})
			})

			r.Get("/report.csv", func(w http.ResponseWriter, req *http.Request) {
				rows := httpx.QueryInt(req, "rows", 25, 1, 10_000)
				attach(w, "report.csv", "text/csv; charset=utf-8")
				w.Write(csvFor(session.MustFromContext(req.Context()), rows))
			})

			r.Get("/slow.csv", func(w http.ResponseWriter, req *http.Request) {
				delay := httpx.QueryInt(req, "ms", generatedDelayMs, 0, 30_000)
				if err := stall(req.Context(), time.Duration(delay)*time.Millisecond); err != nil {
					return
				}
				attach(w, "slow-report.csv", "text/csv; charset=utf-8")
				w.Write(csvFor(session.MustFromContext(req.Context()), 10))
			})

			r.Get("/bundle.zip", func(w http.ResponseWriter, req *http.Request) {
				archive, err := bundleFor(session.MustFromContext(req.Context()))
				if err != nil {
					httpx.Fail(w, http.StatusInternalServerError, "could not build the archive")
					return
				}
				attach(w, "bundle.zip", "application/zip")
				w.Write(archive)
			})

			r.Get("/pixel.png", func(w http.ResponseWriter, req *http.Request) {
				attach(w, "pixel.png", "image/png")
				w.Write(pixelFor(session.MustFromContext(req.Context())))
			})

			// Inline rather than attachment: the browser renders this instead
			// of saving it, which is a different path with a different failure.
			r.Get("/notes.txt", func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("Content-Disposition", `inline; filename="notes.txt"`)
				w.Write([]byte("Served inline. The browser renders this rather than saving it,\nso a download listener never fires.\n"))
			})

			// mime.FormatMediaType emits filename* per RFC 5987 when the name
			// needs it, and a plain ASCII filename beside it for old clients.
			r.Get("/unicode.txt", func(w http.ResponseWriter, _ *http.Request) {
				name := "résumé — 履歴書.txt"
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
					"filename": name,
				}))
				w.Write([]byte("The filename of this file is not ASCII.\n"))
			})
		},
	}
}

func attach(w http.ResponseWriter, name, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
}

// csvFor generates rows from the session's own stream, so the file is
// byte-identical for a given seed and safe to assert on exactly.
func csvFor(sess *session.Session, rows int) []byte {
	stream := sess.RNG.Stream("downloads")

	var buf bytes.Buffer
	out := csv.NewWriter(&buf)
	out.Write([]string{"index", "name", "status", "amount"})

	for i := range rows {
		person := fake.NewPerson(stream, i)
		out.Write([]string{strconv.Itoa(i), person.Name, person.Status, person.Amount})
	}
	out.Flush()
	return buf.Bytes()
}

func bundleFor(sess *session.Session) ([]byte, error) {
	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)

	entries := map[string][]byte{
		"report.csv": csvFor(sess, 5),
		"README.txt": []byte("Two files, one archive, generated from seed " +
			strconv.FormatUint(sess.RNG.Seed(), 10) + ".\n"),
	}
	for _, name := range []string{"README.txt", "report.csv"} {
		entry, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write(entries[name]); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func pixelFor(sess *session.Session) []byte {
	stream := sess.RNG.Stream("downloads-image")
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))

	for y := range 32 {
		for x := range 32 {
			img.Set(x, y, color.RGBA{
				R: uint8(stream.IntN(256)),
				G: uint8(stream.IntN(256)),
				B: uint8(stream.IntN(256)),
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}
