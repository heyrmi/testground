// Package httpx holds the small HTTP helpers the server and every zone share,
// so all of them answer with the same JSON and error shapes.
package httpx

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// Error is the one error body every JSON route returns.
type Error struct {
	Status  int    `json:"status"`
	Message string `json:"error"`
}

// JSON writes a compact response, for payloads a machine reads.
func JSON(w http.ResponseWriter, status int, body any) {
	write(w, status, body, "")
}

// JSONIndent writes an indented response, for payloads people read in a
// browser and diff in a repository.
func JSONIndent(w http.ResponseWriter, status int, body any) {
	write(w, status, body, "  ")
}

// Fail writes an error body.
func Fail(w http.ResponseWriter, status int, message string) {
	JSON(w, status, Error{Status: status, Message: message})
}

func write(w http.ResponseWriter, status int, body any, indent string) {
	encoded, err := marshal(body, indent)
	if err != nil {
		http.Error(w, `{"status":500,"error":"could not encode response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	w.Write(append(encoded, '\n'))
}

func marshal(body any, indent string) ([]byte, error) {
	if indent == "" {
		return json.Marshal(body)
	}
	return json.MarshalIndent(body, "", indent)
}

// QueryInt reads a bounded integer from the query string. A missing or
// unparseable value falls back rather than failing the request, and an
// out-of-range one clamps, so a mistyped URL still yields a working page.
func QueryInt(r *http.Request, name string, fallback, min, max int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return Clamp(parsed, min, max)
}

// Clamp confines v to the inclusive range.
func Clamp(v, min, max int) int {
	switch {
	case v < min:
		return min
	case v > max:
		return max
	default:
		return v
	}
}
