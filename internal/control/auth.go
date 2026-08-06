package control

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/auth"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

// AuthDump is everything a test needs to get through the authentication
// challenges without guessing.
//
// Publishing secrets is the whole point. A two-factor challenge whose codes
// nobody can compute is not a challenge, it is a locked door; a magic-link
// flow whose email nobody can read is the same. Real suites solve this with a
// back channel into the system under test, and this is that back channel,
// made explicit rather than improvised.
type AuthDump struct {
	Users []auth.User `json:"users"`
	// SigningKey is hex, and is what a test signs its own tokens with when it
	// wants an expired or malformed one on purpose.
	SigningKey string `json:"signingKey"`
	TOTPSecret string `json:"totpSecret"`
	TOTPURI    string `json:"totpUri"`
	// TOTPCode is the code valid right now on this session's clock, so a test
	// can submit it without implementing the algorithm.
	TOTPCode string `json:"totpCode"`
	// MagicLinks maps an unredeemed token to the user it signs in.
	MagicLinks map[string]string `json:"magicLinks"`
	Login      *auth.Login       `json:"login"`
	Attempts   int               `json:"failedAttempts"`
	LockedFor  string            `json:"lockedFor,omitempty"`
}

func authRoutes(r chi.Router) {
	r.Get("/auth", func(w http.ResponseWriter, req *http.Request) {
		sess := session.MustFromContext(req.Context())
		httpx.JSONIndent(w, http.StatusOK, authDump(sess))
	})

	// Clearing the login without clearing the rest of the session, which is
	// what a suite wants between tests that share a worker.
	r.Post("/auth/reset", func(w http.ResponseWriter, req *http.Request) {
		sess := session.MustFromContext(req.Context())
		auth.For(sess).Reset()
		httpx.JSONIndent(w, http.StatusOK, authDump(sess))
	})
}

func authDump(sess *session.Session) AuthDump {
	store := auth.For(sess)
	now := sess.Clock.Now()

	code, err := store.TOTPCode(now)
	if err != nil {
		code = ""
	}

	dump := AuthDump{
		Users:      auth.Users,
		SigningKey: store.SecretHex(),
		TOTPSecret: store.TOTPSecret(),
		TOTPURI:    store.TOTPURI("twofactor"),
		TOTPCode:   code,
		MagicLinks: store.PendingMagicLinks(),
		Login:      store.Current(),
	}

	attempts, lockedUntil := store.Attempts()
	dump.Attempts = attempts
	if now.Before(lockedUntil) {
		dump.LockedFor = lockedUntil.Sub(now).Round(0).String()
	}
	return dump
}
