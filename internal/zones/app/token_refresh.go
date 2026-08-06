package app

import (
	"net/http"
	"strings"

	"github.com/heyrmi/testground/internal/auth"
	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/session"
)

func tokenRefresh() challenge.Challenge {
	return challenge.Challenge{
		ID:       "token-refresh",
		Title:    "An access token that expires mid-suite",
		URL:      "/app/token-refresh",
		Zone:     challenge.ZoneApp,
		Tier:     challenge.T3,
		Category: "K. Authentication",
		Summary: "Signing in returns an access token good for sixty seconds and a refresh " +
			"token good for thirty minutes. A protected endpoint answers 401 the moment " +
			"the access token expires, and the page can be told whether to refresh " +
			"automatically or to let the failure through.",
		WhyHard: "Sixty seconds is invisible to one test and unavoidable for a suite. The " +
			"failure lands on whichever test happens to be running when the token runs " +
			"out, which is a different test every run -- so it reads as flake in an " +
			"unrelated feature rather than as an auth problem, and the usual response is " +
			"to add a retry to a test that was never broken. Expiry is also easy to " +
			"confuse with rejection: a wrong token and an old token both give 401 unless " +
			"something distinguishes them, and only one of them is fixed by refreshing.",
		Hint: "Do not wait sixty seconds. The token expires on the session clock, so " +
			"advancing it through the control plane makes the expiry happen now, every " +
			"time, in the same place. The endpoint says which kind of 401 it gave, so a " +
			"test can prove the refresh path ran rather than inferring it from a request " +
			"that eventually worked. The signing key is published, so a token that is " +
			"expired or malformed on purpose is a thing you can construct rather than " +
			"wait for.",
		Tags:     []string{"auth", "jwt", "refresh", "expiry", "401"},
		Concepts: []string{"short-lived tokens fail across a suite", "expired is not the same as invalid", "refresh flows", "moving the clock instead of waiting"},
		Selectors: []challenge.Selector{
			{TestID: "sign-in", Role: "button", Note: "Signs in and takes both tokens"},
			{TestID: "call-api", Role: "button", Note: "Calls the protected endpoint once"},
			{TestID: "auto-refresh", Role: "checkbox", Note: "Whether a 401 is retried after refreshing"},
			{TestID: "token-state", Note: "none, valid or expired, as the page last saw it"},
			{TestID: "last-status", Note: "The status the protected endpoint last gave"},
			{TestID: "last-reason", Note: "Which kind of 401 it was, when it was one"},
			{TestID: "refresh-count", Note: "How many times the page refreshed the access token"},
			{TestID: "identity", Transient: true, Note: "Who the protected endpoint says you are"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/api/app/auth/login", Note: "Returns an access and a refresh token"},
			{Method: http.MethodGet, Path: "/api/app/auth/me", Note: "Needs a bearer access token; 401 says which kind"},
			{Method: http.MethodPost, Path: "/api/app/auth/refresh", Note: "Exchanges a refresh token for a new access token"},
			{Method: http.MethodGet, Path: "/api/control/auth", Note: "The signing key, for building tokens deliberately"},
		},
		Stability: challenge.Stable,
	}
}

type tokenPair struct {
	Access    string `json:"access"`
	Refresh   string `json:"refresh"`
	ExpiresIn int    `json:"expiresInSeconds"`
	Username  string `json:"username"`
	Role      string `json:"role"`
}

func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	store := auth.For(sess)

	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	decodeJSON(r, &body)
	if body.Username == "" {
		body.Username, body.Password = "user", "user123"
	}

	now := sess.Clock.Now()
	login, throttled, ok := store.Attempt(now, body.Username, body.Password)
	switch {
	case throttled:
		httpx.Fail(w, http.StatusTooManyRequests, "too many attempts")
		return
	case !ok:
		httpx.Fail(w, http.StatusUnauthorized, "bad credentials")
		return
	}

	access, err := store.Issue(now, *login, "access")
	refresh, err2 := store.Issue(now, *login, "refresh")
	if err != nil || err2 != nil {
		httpx.Fail(w, http.StatusInternalServerError, "could not issue tokens")
		return
	}

	httpx.JSON(w, http.StatusOK, tokenPair{
		Access:    access,
		Refresh:   refresh,
		ExpiresIn: int(auth.AccessLifetime.Seconds()),
		Username:  login.Username,
		Role:      login.Role,
	})
}

// The 401 says which kind it is. Without that a test cannot tell an expired
// token from a wrong one, and only one of the two is fixed by refreshing.
func handleAuthMe(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	store := auth.For(sess)

	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if token == "" {
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{
			"status": http.StatusUnauthorized, "error": "no token", "reason": "missing",
		})
		return
	}

	claims, err := store.Verify(sess.Clock.Now(), token)
	switch {
	case err == auth.ErrExpired:
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{
			"status": http.StatusUnauthorized, "error": "token expired", "reason": "expired",
		})
		return
	case err != nil:
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{
			"status": http.StatusUnauthorized, "error": "token rejected", "reason": "invalid",
		})
		return
	case claims.Kind != "access":
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{
			"status": http.StatusUnauthorized, "error": "not an access token", "reason": "wrong-kind",
		})
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"username": claims.Username,
		"role":     claims.Role,
		"tokenId":  claims.ID,
	})
}

func handleAuthRefresh(w http.ResponseWriter, r *http.Request) {
	sess := session.MustFromContext(r.Context())
	store := auth.For(sess)

	var body struct {
		Refresh string `json:"refresh"`
	}
	decodeJSON(r, &body)

	now := sess.Clock.Now()
	claims, err := store.Verify(now, body.Refresh)
	if err != nil || claims.Kind != "refresh" {
		reason := "invalid"
		if err == auth.ErrExpired {
			reason = "expired"
		}
		httpx.JSON(w, http.StatusUnauthorized, map[string]any{
			"status": http.StatusUnauthorized, "error": "refresh rejected", "reason": reason,
		})
		return
	}

	access, err := store.Issue(now, auth.Login{Username: claims.Username, Role: claims.Role}, "access")
	if err != nil {
		httpx.Fail(w, http.StatusInternalServerError, "could not issue a token")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"access":           access,
		"expiresInSeconds": int(auth.AccessLifetime.Seconds()),
	})
}
