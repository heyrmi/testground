// Package auth is the identity machinery the authentication challenges share.
//
// Everything here is per session. Two workers logging in as the same user get
// two independent logins, two independent signing keys and two independent
// lockout counters, so a suite can exercise a rate limit in one worker without
// locking the others out.
//
// Nothing here is a security control. The signing key is derived from the
// session seed on purpose, because a test that cannot predict the key cannot
// forge the expired token it needs to exercise the expiry path. Every secret
// this package holds is meant to be readable through the control plane.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/heyrmi/testground/internal/session"
)

// Key is the session state key the auth store lives under.
const Key = "auth"

// MaxAttempts is how many failed logins a session gets before it is throttled.
const MaxAttempts = 5

// Lockout is how long the throttle lasts, measured on the session clock so a
// test can advance past it rather than waiting.
const Lockout = 30 * time.Second

// User is one of the fixed accounts every session starts with.
type User struct {
	Username string `json:"username"`
	Password string `json:"-"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	// TwoFactor means the password alone is not enough.
	TwoFactor bool `json:"twoFactor"`
}

// Users are the same in every session, and their passwords are printed on the
// pages that use them. Guessing credentials is not the exercise.
var Users = []User{
	{Username: "admin", Password: "admin123", Role: "admin", Name: "Ada Admin"},
	{Username: "user", Password: "user123", Role: "user", Name: "Uma User"},
	{Username: "viewer", Password: "viewer123", Role: "viewer", Name: "Vic Viewer"},
	{Username: "twofactor", Password: "twofactor123", Role: "user", Name: "Tam Two-Factor", TwoFactor: true},
}

// Lookup finds a fixed account by name.
func Lookup(username string) (User, bool) {
	for _, user := range Users {
		if strings.EqualFold(user.Username, username) {
			return user, true
		}
	}
	return User{}, false
}

// Login is a session's current authenticated state.
type Login struct {
	Username string    `json:"username"`
	Role     string    `json:"role"`
	Name     string    `json:"name"`
	At       time.Time `json:"at"`
	// Pending means the password was right and the second factor is not in yet.
	Pending bool `json:"pending"`
	// Remembered marks a login that survives a browser restart, which here
	// means it survives the session cookie being cleared.
	Remembered bool `json:"remembered"`
}

// Store holds one session's authentication state.
type Store struct {
	// secret keys everything this session signs. Derived from the seed, so a
	// test can reproduce and forge tokens deliberately.
	secret []byte
	// totpSecret is the shared secret for the two-factor account, published
	// through the control plane so tests can compute real codes.
	totpSecret string

	// sessionID scopes the CSRF token. The signing key is seed-derived on
	// purpose so tests can forge tokens; a CSRF token that is the same in
	// every session with the same seed would make the whole guard cosmetic.
	sessionID string

	mu          sync.Mutex
	login       *Login
	csrf        string
	attempts    int
	lockedUntil time.Time
	magicTokens map[string]string
	// issued names the tokens minted for the login that is current now. Without
	// it the store cannot say which ids belong to the session it is ending, and
	// revocation could only ever be driven by hand from a test.
	issued  map[string]bool
	revoked map[string]bool
}

// For returns the session's auth store, creating it on first use.
func For(sess *session.Session) *Store {
	return session.Value(sess, Key, func() *Store {
		stream := sess.RNG.Stream("auth")

		secret := make([]byte, 32)
		for i := range secret {
			secret[i] = byte(stream.IntN(256))
		}

		// A twenty-byte TOTP secret, base32 encoded as the standard expects.
		raw := make([]byte, 20)
		for i := range raw {
			raw[i] = byte(stream.IntN(256))
		}

		return &Store{
			sessionID:   string(sess.ID),
			secret:      secret,
			totpSecret:  base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw),
			magicTokens: make(map[string]string),
			issued:      make(map[string]bool),
			revoked:     make(map[string]bool),
		}
	})
}

// Secret is the session's signing key, in hex. Published deliberately: a test
// that cannot sign a token cannot build the expired one it needs.
func (s *Store) Secret() []byte { return s.secret }

// SecretHex is the signing key in a form a test can read and use.
func (s *Store) SecretHex() string { return hex.EncodeToString(s.secret) }

// TOTPSecret is the shared secret for the two-factor account.
func (s *Store) TOTPSecret() string { return s.totpSecret }

// Current reports who is logged in, if anyone.
func (s *Store) Current() *Login {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.login == nil {
		return nil
	}
	copied := *s.login
	return &copied
}

// Attempt tries a password. It reports the login on success, and whether the
// session is throttled, so the caller can tell "wrong password" from "too many
// wrong passwords" -- which are different messages to a user and different
// statuses to a test.
func (s *Store) Attempt(now time.Time, username, password string) (login *Login, throttled bool, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if now.Before(s.lockedUntil) {
		return nil, true, false
	}

	user, found := Lookup(username)
	if !found || user.Password != password {
		s.attempts++
		if s.attempts >= MaxAttempts {
			s.lockedUntil = now.Add(Lockout)
		}
		return nil, false, false
	}

	s.attempts = 0
	s.lockedUntil = time.Time{}
	s.login = &Login{
		Username: user.Username,
		Role:     user.Role,
		Name:     user.Name,
		At:       now,
		Pending:  user.TwoFactor,
	}

	copied := *s.login
	return &copied, false, true
}

// CompleteSecondFactor promotes a pending login to a complete one.
func (s *Store) CompleteSecondFactor() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.login == nil || !s.login.Pending {
		return false
	}
	s.login.Pending = false
	return true
}

// Remember marks the login as surviving a browser restart.
func (s *Store) Remember(remembered bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.login != nil {
		s.login.Remembered = remembered
	}
}

// SignIn logs a user in directly, for the flows that do not use a password.
func (s *Store) SignIn(now time.Time, user User) *Login {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.login = &Login{Username: user.Username, Role: user.Role, Name: user.Name, At: now}
	copied := *s.login
	return &copied
}

// LogOut ends the login on the server, which is the part that matters: a
// cookie a client keeps is worth nothing once the server has forgotten it.
//
// The same has to be true of a bearer token, and a signed token is otherwise
// self-contained -- nothing about verifying one reaches back to the server. So
// every id minted for the login being ended is withdrawn here, which is what
// stops an access token taken before the logout from outliving it.
func (s *Store) LogOut() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.login = nil
	for id := range s.issued {
		s.revoked[id] = true
	}
	s.issued = make(map[string]bool)
}

// Attempts reports failed logins since the last success, and when the throttle
// lifts.
func (s *Store) Attempts() (count int, lockedUntil time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempts, s.lockedUntil
}

// CSRF returns the session's token, minting one on first use. It is stable for
// the session rather than rotating per form, which keeps the challenge about
// sending the token at all rather than about chasing it.
func (s *Store) CSRF() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.csrf == "" {
		mac := hmac.New(sha256.New, s.secret)
		fmt.Fprintf(mac, "csrf:%s", s.sessionID)
		s.csrf = hex.EncodeToString(mac.Sum(nil))[:32]
	}
	return s.csrf
}

// IssueMagicToken mints a single-use sign-in token for a user.
func (s *Store) IssueMagicToken(username string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "magic:%s:%d", username, len(s.magicTokens))
	token := hex.EncodeToString(mac.Sum(nil))[:24]

	s.magicTokens[token] = username
	return token
}

// RedeemMagicToken consumes a token, reporting who it belonged to. A token
// works once, which is the whole point of the pattern.
func (s *Store) RedeemMagicToken(token string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username, ok := s.magicTokens[token]
	if ok {
		delete(s.magicTokens, token)
	}
	return username, ok
}

// PendingMagicLinks lists the tokens not yet redeemed, for the control plane
// to publish. Fetching the link out of band is how a test opens an email it
// cannot read.
func (s *Store) PendingMagicLinks() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make(map[string]string, len(s.magicTokens))
	for token, username := range s.magicTokens {
		out[token] = username
	}
	return out
}

// Revoke marks a token id as no longer acceptable, which is what makes logging
// out of a stateless token scheme mean anything.
func (s *Store) Revoke(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[id] = true
}

// noteIssued records an id against the current login so LogOut has something to
// withdraw.
//
// It also lifts any earlier revocation of that id, which is not as odd as it
// reads: an id carries the instant it was minted at, and the session clock can
// be frozen, so signing in again at a standstill reproduces a token byte for
// byte. Leaving it revoked would hand a caller a token that was dead on
// arrival, with no way to tell it apart from the one the logout killed.
func (s *Store) noteIssued(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.issued[id] = true
	delete(s.revoked, id)
}

// Revoked reports whether a token id has been withdrawn.
func (s *Store) Revoked(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revoked[id]
}

// Reset clears the login, the throttle and every issued token.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.login = nil
	s.attempts = 0
	s.lockedUntil = time.Time{}
	s.magicTokens = make(map[string]string)
	s.issued = make(map[string]bool)
	s.revoked = make(map[string]bool)
}
