package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io/fs"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

// RequestIDHeader lets a failing test correlate a response with a log line.
const RequestIDHeader = "X-Request-Id"

type requestIDKey struct{}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(RequestIDHeader)
		if id == "" {
			buf := make([]byte, 8)
			rand.Read(buf)
			id = hex.EncodeToString(buf)
		}
		w.Header().Set(RequestIDHeader, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// recorder captures what was written so the log line can report it. It
// deliberately implements only WriteHeader and Write: challenges need flushing
// and hijacking, so the unwrapping helpers below expose the original writer.
type recorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rec *recorder) WriteHeader(status int) {
	if rec.status == 0 {
		rec.status = status
		rec.ResponseWriter.WriteHeader(status)
	}
}

func (rec *recorder) Write(p []byte) (int, error) {
	if rec.status == 0 {
		rec.status = http.StatusOK
	}
	n, err := rec.ResponseWriter.Write(p)
	rec.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the flusher and hijacker
// underneath, which the streaming and realtime challenges rely on.
func (rec *recorder) Unwrap() http.ResponseWriter { return rec.ResponseWriter }

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		rec := &recorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		if rec.status == 0 {
			rec.status = http.StatusOK
		}
		s.log.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"duration", time.Since(started).Round(time.Microsecond).String(),
			"session", rec.Header().Get(session.Header),
			"request_id", requestIDFrom(r.Context()),
		)
	})
}

// recoverPanics keeps one broken challenge from taking the server down, which
// matters because the whole point is to run deliberately hostile pages.
func (s *Server) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil || recovered == http.ErrAbortHandler {
				return
			}
			s.log.ErrorContext(r.Context(), "panic serving request",
				"path", r.URL.Path,
				"panic", recovered,
				"stack", string(debug.Stack()),
				"request_id", requestIDFrom(r.Context()),
			)
			httpx.Fail(w, http.StatusInternalServerError, "the playground panicked serving this route")
		}()
		next.ServeHTTP(w, r)
	})
}

// cacheableFileServer serves embedded assets. Embedded files have no
// modification time, so revalidation is disabled rather than left to a
// Last-Modified header the FS cannot supply.
func cacheableFileServer(fsys fs.FS) http.Handler {
	server := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		server.ServeHTTP(w, r)
	})
}
