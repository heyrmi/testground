package control

import (
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

// Prefix is where the control plane is mounted. Rules never apply to requests
// under it: a failure rule matching everything must not be able to lock an
// operator out of the API that would remove it.
const Prefix = "/api/control"

// For returns the control surface belonging to a session, creating it on first
// use.
func For(sess *session.Session) *State {
	return session.Value(sess, Key, func() *State { return New(sess.RNG) })
}

// Routes serves the control plane.
func Routes(store *session.Store) http.Handler {
	r := chi.NewRouter()

	r.Post("/session", func(w http.ResponseWriter, req *http.Request) {
		created := store.Create()
		httpx.JSONIndent(w, http.StatusCreated, map[string]any{
			"session": string(created.ID),
			"seed":    created.RNG.Seed(),
			"header":  session.Header,
		})
	})

	r.Delete("/session/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		if !session.ValidID(id) {
			httpx.Fail(w, http.StatusBadRequest, "invalid session id")
			return
		}
		httpx.JSONIndent(w, http.StatusOK, map[string]any{
			"session": id,
			"deleted": store.Delete(session.ID(id)),
		})
	})

	// Everything below acts on the caller's own session, which is what makes
	// parallel workers able to drive their own copy without coordinating.
	r.Post("/reset", func(w http.ResponseWriter, req *http.Request) {
		sess := session.MustFromContext(req.Context())
		For(sess).Reset()
		sess.Reset()
		writeState(w, sess)
	})

	r.Post("/seed", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Seed *uint64 `json:"seed"`
		}
		if !decode(w, req, &body) {
			return
		}
		if body.Seed == nil {
			httpx.Fail(w, http.StatusBadRequest, "seed is required")
			return
		}

		sess := session.MustFromContext(req.Context())
		// Reseeding drops derived state, including the control surface, since
		// rules replaying against the old seed would no longer be reproducible.
		sess.Reseed(*body.Seed)
		writeState(w, sess)
	})

	r.Post("/latency", func(w http.ResponseWriter, req *http.Request) {
		var rule LatencyRule
		if !decode(w, req, &rule) {
			return
		}
		if rule.Route == "" {
			httpx.Fail(w, http.StatusBadRequest, "route is required")
			return
		}

		sess := session.MustFromContext(req.Context())
		For(sess).SetLatency(rule)
		writeState(w, sess)
	})

	r.Post("/failure", func(w http.ResponseWriter, req *http.Request) {
		var rule FailureRule
		if !decode(w, req, &rule) {
			return
		}
		switch {
		case rule.Route == "":
			httpx.Fail(w, http.StatusBadRequest, "route is required")
			return
		case rule.Rate < 0 || rule.Rate > 1:
			httpx.Fail(w, http.StatusBadRequest, "rate must be between 0 and 1")
			return
		}

		sess := session.MustFromContext(req.Context())
		For(sess).SetFailure(rule)
		writeState(w, sess)
	})

	r.Post("/flake", func(w http.ResponseWriter, req *http.Request) {
		var rule FlakeRule
		if !decode(w, req, &rule) {
			return
		}
		switch {
		case rule.Challenge == "":
			httpx.Fail(w, http.StatusBadRequest, "challenge is required")
			return
		case rule.Probability < 0 || rule.Probability > 1:
			httpx.Fail(w, http.StatusBadRequest, "probability must be between 0 and 1")
			return
		}

		sess := session.MustFromContext(req.Context())
		For(sess).SetFlake(rule)
		writeState(w, sess)
	})

	r.Post("/clock", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Action  string `json:"action"`
			Ms      int64  `json:"ms,omitempty"`
			Instant string `json:"instant,omitempty"`
		}
		if !decode(w, req, &body) {
			return
		}

		sess := session.MustFromContext(req.Context())
		switch body.Action {
		case "freeze":
			sess.Clock.Freeze()
		case "unfreeze":
			sess.Clock.Unfreeze()
		case "advance":
			sess.Clock.Advance(time.Duration(body.Ms) * time.Millisecond)
		case "set":
			instant, err := time.Parse(time.RFC3339, body.Instant)
			if err != nil {
				httpx.Fail(w, http.StatusBadRequest, "instant must be RFC 3339")
				return
			}
			sess.Clock.Set(instant)
		case "reset":
			sess.Clock.Reset()
		default:
			httpx.Fail(w, http.StatusBadRequest, "action must be freeze, unfreeze, advance, set or reset")
			return
		}
		writeState(w, sess)
	})

	r.Post("/feature", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Flag    string `json:"flag"`
			Enabled bool   `json:"enabled"`
		}
		if !decode(w, req, &body) {
			return
		}
		if body.Flag == "" {
			httpx.Fail(w, http.StatusBadRequest, "flag is required")
			return
		}

		sess := session.MustFromContext(req.Context())
		For(sess).SetFeature(body.Flag, body.Enabled)
		writeState(w, sess)
	})

	r.Get("/state", func(w http.ResponseWriter, req *http.Request) {
		writeState(w, session.MustFromContext(req.Context()))
	})

	return r
}

// Dump is the assertion target: everything a test might want to check about
// the state its own session is in.
type Dump struct {
	Session string     `json:"session"`
	Seed    uint64     `json:"seed"`
	Clock   ClockState `json:"clock"`
	Control Snapshot   `json:"control"`
	// Challenges lists the challenge state keys this session has touched, so a
	// test can see whether a page has been interacted with at all. The control
	// surface itself is not one of them.
	Challenges []string `json:"challenges"`
}

// ClockState reports what the session's clock currently reads.
type ClockState struct {
	Now    time.Time `json:"now"`
	Frozen bool      `json:"frozen"`
}

func writeState(w http.ResponseWriter, sess *session.Session) {
	httpx.JSONIndent(w, http.StatusOK, Dump{
		Session:    string(sess.ID),
		Seed:       sess.RNG.Seed(),
		Clock:      ClockState{Now: sess.Clock.Now().UTC(), Frozen: sess.Clock.Frozen()},
		Control:    For(sess).Snapshot(),
		Challenges: challengeKeys(sess),
	})
}

func decode(w http.ResponseWriter, r *http.Request, into any) bool {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		httpx.Fail(w, http.StatusBadRequest, "could not read the request body")
		return false
	}
	if len(body) == 0 {
		httpx.Fail(w, http.StatusBadRequest, "a JSON body is required")
		return false
	}
	if err := json.Unmarshal(body, into); err != nil {
		httpx.Fail(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

// challengeKeys lists the session's state keys with the control surface's own
// entry removed, since the control plane is not a challenge.
func challengeKeys(sess *session.Session) []string {
	keys := sess.Keys()
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		if key != Key {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}
