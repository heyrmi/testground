package control

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

// HeaderInjected marks a response the control plane interfered with, so a
// test reading a 503 can tell an injected one from a real one.
const HeaderInjected = "X-Playground-Injected"

// Middleware applies the caller's own latency and failure rules.
//
// It never applies them to the control plane itself. A failure rule matching
// every route is a reasonable thing to want, and it must not take away the
// ability to remove it.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.FromContext(r.Context())
		if sess == nil || strings.HasPrefix(r.URL.Path, Prefix) {
			next.ServeHTTP(w, r)
			return
		}

		state := For(sess)

		if ms := state.LatencyFor(r.URL.Path); ms > 0 {
			if !sleep(r, time.Duration(ms)*time.Millisecond) {
				return // the client gave up while being delayed
			}
		}

		if status, message, fail := state.FailureFor(r.URL.Path); fail {
			if message == "" {
				message = "injected failure for " + r.URL.Path
			}
			w.Header().Set(HeaderInjected, strconv.Itoa(status))
			httpx.Fail(w, status, message)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func sleep(r *http.Request, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-r.Context().Done():
		return false
	case <-timer.C:
		return true
	}
}
