package classic

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/auth"
	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

type loginView struct {
	Login     *auth.Login
	CSRF      string
	Users     []auth.User
	Error     string
	Attempts  int
	LockedFor string
}

func formLogin() page {
	meta := challenge.Challenge{
		ID:       "form-login",
		Title:    "Log in, and everything that guards it",
		URL:      "/classic/form-login",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T2,
		Category: "K. Authentication",
		Summary: "A password form with a CSRF token, a throttle that refuses the sixth wrong " +
			"attempt for thirty seconds, a remember-me box, and a logout that ends the " +
			"login on the server rather than only in the browser. Credentials are printed " +
			"on the page.",
		WhyHard: "The three ways this refuses you look similar and mean different things. A " +
			"wrong password is 200 with a message, a missing or stale CSRF token is 403 " +
			"before the password is even considered, and a throttled session is 429 no " +
			"matter how right the password is -- so a test that only checks 'am I logged " +
			"in' cannot tell a credential bug from a token bug from its own earlier " +
			"failures. The throttle is the sharp edge: a suite that exercises bad " +
			"passwords locks its own worker out of every test that runs after it.",
		Hint: "Assert on the status as well as the page. Take the CSRF token from the form " +
			"rather than hard-coding one; it is per session, and a copied token from " +
			"another worker is a 403. The throttle is measured on the session clock, so " +
			"advance it through the control plane rather than waiting, and reset the auth " +
			"state between tests that deliberately fail -- there is an endpoint for exactly " +
			"that. Logging out revokes the login on the server, so keeping the cookie " +
			"afterwards buys nothing.",
		Tags:     []string{"auth", "csrf", "rate-limit", "session", "logout"},
		Concepts: []string{"CSRF tokens are per session", "throttles outlive the test that caused them", "server-side logout", "status codes distinguish failures"},
		Selectors: []challenge.Selector{
			{TestID: "login-form", Note: "Carries the CSRF token in a hidden field"},
			{TestID: "field-username", Role: "textbox", Note: "Credentials are printed beside the form"},
			{TestID: "field-password", Role: "textbox", Note: "Password field"},
			{TestID: "field-remember", Role: "checkbox", Note: "Marks the login as surviving a restart"},
			{TestID: "csrf-token", Note: "The hidden token; read it rather than inventing one"},
			{TestID: "submit", Role: "button", Note: "Posts the form"},
			{TestID: "login-error", Transient: true, Note: "Why the last attempt was refused"},
			{TestID: "attempts", Note: "Failed attempts since the last success"},
			{TestID: "welcome", Transient: true, Note: "Shown once logged in; carries the name and role"},
			{TestID: "current-role", Transient: true, Note: "The role the server believes you have"},
			{TestID: "remembered", Transient: true, Note: "Present when the login was marked remembered"},
			{TestID: "logout", Role: "button", Transient: true, Note: "Ends the login on the server"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/form-login", Note: "Needs the CSRF token; answers 403, 429 or 303"},
			{Method: http.MethodPost, Path: "/classic/form-login/logout", Note: "Ends the server-side login"},
			{Method: http.MethodGet, Path: "/api/control/auth", Note: "Credentials, throttle state and the current login"},
			{Method: http.MethodPost, Path: "/api/control/auth/reset", Note: "Clears the login and the throttle between tests"},
		},
		Stability: challenge.Stable,
	}

	view := func(req *http.Request, message string) render.View {
		sess := session.MustFromContext(req.Context())
		store := auth.For(sess)
		attempts, lockedUntil := store.Attempts()

		data := loginView{
			Login:    store.Current(),
			CSRF:     store.CSRF(),
			Users:    auth.Users,
			Error:    message,
			Attempts: attempts,
		}
		if now := sess.Clock.Now(); now.Before(lockedUntil) {
			data.LockedFor = lockedUntil.Sub(now).Round(time.Second).String()
		}
		return render.View{Title: meta.Title, Challenge: &meta, Data: data}
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/form-login", view(req, ""))
			})

			r.Post("/", func(w http.ResponseWriter, req *http.Request) {
				if err := req.ParseForm(); err != nil {
					renderer.PageStatus(w, req, http.StatusBadRequest, "classic/form-login",
						view(req, "the form could not be read"))
					return
				}

				sess := session.MustFromContext(req.Context())
				store := auth.For(sess)

				// Checked before the password, so a missing token is never
				// mistaken for a credential problem.
				if req.PostFormValue("csrf") != store.CSRF() {
					renderer.PageStatus(w, req, http.StatusForbidden, "classic/form-login",
						view(req, "the CSRF token was missing or did not match this session"))
					return
				}

				login, throttled, ok := store.Attempt(
					sess.Clock.Now(), req.PostFormValue("username"), req.PostFormValue("password"))

				switch {
				case throttled:
					renderer.PageStatus(w, req, http.StatusTooManyRequests, "classic/form-login",
						view(req, "too many attempts; this session is throttled"))
				case !ok:
					renderer.PageStatus(w, req, http.StatusOK, "classic/form-login",
						view(req, "that username and password do not match"))
				default:
					if login.Pending {
						store.CompleteSecondFactor() // this page does not ask for one
					}
					store.Remember(req.PostFormValue("remember") != "")
					http.Redirect(w, req, meta.URL, http.StatusSeeOther)
				}
			})

			r.Post("/logout", func(w http.ResponseWriter, req *http.Request) {
				auth.For(session.MustFromContext(req.Context())).LogOut()
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})
		},
	}
}
