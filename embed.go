// Package testground holds the assets compiled into the binary.
//
// The embed directives live at the module root because go:embed cannot reach
// above its own package directory, and the asset trees are shared by several
// internal packages.
package testground

import (
	"embed"
	"io/fs"
)

//go:embed templates
var templates embed.FS

//go:embed web/static
var static embed.FS

// The built frontend is committed so `go install` yields a working binary
// rather than one with an empty /app. all: keeps dotfiles so the tree embeds
// even before anyone has run the Vite build.
//
//go:embed all:web/app/dist
var appDist embed.FS

// Templates is the Go template tree, rooted at the templates directory.
func Templates() fs.FS { return must(fs.Sub(templates, "templates")) }

// Static is the vendored stylesheet and script tree served under /static.
// Nothing here is fetched from a CDN: the playground must work offline.
func Static() fs.FS { return must(fs.Sub(static, "web/static")) }

// AppDist is the Vite build output for the SPA and web component zones.
func AppDist() fs.FS { return must(fs.Sub(appDist, "web/app/dist")) }

func must(sub fs.FS, err error) fs.FS {
	if err != nil {
		panic("testground: embedded assets are missing: " + err.Error())
	}
	return sub
}
