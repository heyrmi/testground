package classic

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/heyrmi/testground/internal/challenge"
)

const (
	// uploadMaxBytes is enforced by the server after the whole file has
	// arrived, which is the point: the transfer completes and then fails.
	uploadMaxBytes  = 64 * 1024
	uploadMaxMemory = 8 << 20
)

var acceptedExtensions = []string{".png", ".jpg", ".jpeg", ".txt", ".csv"}

type uploadedFile struct {
	Field    string
	Name     string
	Size     int64
	Type     string
	Rejected string
}

type uploadValues struct {
	Files    []uploadedFile
	Accepted int
	Rejected int
}

func uploads() page {
	meta := challenge.Challenge{
		ID:       "uploads",
		Title:    "File uploads and the rules that are not enforced",
		URL:      "/classic/uploads",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "N. Files",
		Summary: "A single-file input, a multiple-file input, and one carrying an accept " +
			"attribute. The server rejects anything over 64 kB or with an extension it " +
			"does not recognise, and reports every file it received either way.",
		WhyHard: "A file input cannot be typed into or clicked into a state. The only route " +
			"is the framework's file-setting API, because clicking it opens an " +
			"operating-system picker that no browser automation can drive. The accept " +
			"attribute filters that picker and does nothing else -- a file set " +
			"programmatically, or dropped in, ignores it completely -- so client-side type " +
			"checking is advisory and only the server's answer is real. The size limit is " +
			"worse: it applies after the upload has finished, so a large file spends its " +
			"whole transfer time being accepted and is then refused.",
		Hint: "Set the files through your framework's API rather than trying to click the " +
			"input. Then assert on what the server reported, not on what accept implied " +
			"would be impossible -- this page will happily receive a file that attribute " +
			"claims to exclude. The response arrives after a redirect, so wait for the " +
			"page rather than for the request.",
		Tags:     []string{"files", "upload", "multipart", "validation"},
		Concepts: []string{"file inputs cannot be clicked", "accept is advisory", "server-side validation is the real one", "size limits apply after transfer"},
		Selectors: []challenge.Selector{
			{TestID: "form", Note: "The multipart form the files post through"},
			{TestID: "file-single", Note: "One file at a time"},
			{TestID: "file-multiple", Note: "Accepts several files in one go"},
			{TestID: "file-restricted", Note: "Carries accept, which filters the picker and nothing else"},
			{TestID: "submit", Role: "button", Note: "Posts the multipart form"},
			{TestID: "upload-row", Transient: true, Note: "One per file the server received"},
			{TestID: "accepted-count", Transient: true, Note: "How many files were kept"},
			{TestID: "rejected-count", Transient: true, Note: "How many were refused, and why is on the row"},
			{TestID: "no-submission", Note: "Shown before anything has been uploaded"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/uploads", Note: "Accepts multipart/form-data, answers 303"},
		},
		Stability: challenge.Stable,
	}

	return uploadPage(meta, uploadMaxMemory, func(r *http.Request) uploadValues {
		values := uploadValues{}
		if r.MultipartForm == nil {
			return values
		}

		for _, field := range []string{"single", "multiple", "restricted"} {
			for _, header := range r.MultipartForm.File[field] {
				file := inspect(field, header)
				if file.Rejected == "" {
					values.Accepted++
				} else {
					values.Rejected++
				}
				values.Files = append(values.Files, file)
			}
		}
		return values
	})
}

func inspect(field string, header *multipart.FileHeader) uploadedFile {
	file := uploadedFile{
		Field: field,
		Name:  filepath.Base(header.Filename),
		Size:  header.Size,
		Type:  header.Header.Get("Content-Type"),
	}

	switch {
	case header.Size > uploadMaxBytes:
		file.Rejected = fmt.Sprintf("larger than %d bytes, and the whole file had to arrive before anyone could tell", uploadMaxBytes)
	case !accepted(file.Name):
		file.Rejected = "extension not in " + strings.Join(acceptedExtensions, ", ")
	}
	return file
}

func accepted(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	for _, allowed := range acceptedExtensions {
		if ext == allowed {
			return true
		}
	}
	return false
}
