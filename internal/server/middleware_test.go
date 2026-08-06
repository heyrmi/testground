package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// The realtime and streaming challenges depend on flushing through whatever
// middleware wraps them, so the logging wrapper must stay transparent to
// http.ResponseController. This is the test that keeps it that way.
func TestLoggingWrapperStaysFlushable(t *testing.T) {
	srv := &Server{opts: Options{}, log: discardLogger()}

	flushed := false
	handler := srv.logRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("first chunk"))
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Errorf("flush through the wrapper failed: %v", err)
			return
		}
		flushed = true
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/stream", nil))

	if !flushed {
		t.Fatal("handler could not flush")
	}
	if !rec.Flushed {
		t.Fatal("the flush did not reach the underlying writer")
	}
}

func TestLoggingWrapperRecordsTheStatusItSaw(t *testing.T) {
	srv := &Server{opts: Options{}, log: discardLogger()}

	var seen *recorder
	handler := srv.logRequests(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		seen = w.(*recorder)
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("nope"))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if seen.status != http.StatusTeapot {
		t.Fatalf("recorded status %d, want 418", seen.status)
	}
	if seen.bytes != 4 {
		t.Fatalf("recorded %d bytes, want 4", seen.bytes)
	}
}
