package classic

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/auth"
	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

type twoFactorView struct {
	Login *auth.Login
	CSRF  string
	Error string
	// MagicLinks are the unredeemed sign-in links, shown on the page as the
	// stand-in for an inbox nobody can open.
	MagicLinks map[string]string
}

func twoFactor() page {
	meta := challenge.Challenge{
		ID:       "two-factor",
		Title:    "A second factor, and a link sent somewhere you cannot read",
		URL:      "/classic/two-factor",
		Zone:     challenge.ZoneClassic,
		Tier:     challenge.T3,
		Category: "K. Authentication",
		Summary: "An account whose password is only the first half: it leaves the login " +
			"pending until a six-digit time-based code arrives. Beside it, a sign-in link " +
			"that would normally be emailed, and is instead retrievable through the " +
			"control plane.",
		WhyHard: "Neither of these can be completed from the page alone, and that is the " +
			"point rather than an oversight. The code changes every thirty seconds and " +
			"depends on the clock, so it cannot be written into a fixture -- a hard-coded " +
			"code is wrong within half a minute of being recorded. The link would arrive " +
			"in an inbox no browser test can open. Both are the shape of every real " +
			"authentication flow that defeats a suite, and both are solved the same way: " +
			"a back channel into the system under test.",
		Hint: "The control plane is the back channel. It publishes the shared secret, the " +
			"code valid on this session's clock right now, and every unredeemed sign-in " +
			"link. Read the code from there rather than computing it, or compute it " +
			"yourself from the secret if you want to prove your own implementation. " +
			"Remember the code is tied to the clock: freeze or advance time and the valid " +
			"code moves with it. A link works exactly once.",
		Tags:     []string{"auth", "totp", "two-factor", "magic-link", "back-channel"},
		Concepts: []string{"time-based codes cannot be fixtures", "out-of-band flows need a back channel", "single-use tokens", "codes follow the session clock"},
		Selectors: []challenge.Selector{
			{TestID: "login-form", Note: "Shown while nobody is signed in"},
			{TestID: "field-username", Role: "textbox", Note: "Use the twofactor account"},
			{TestID: "field-password", Role: "textbox", Note: "Its password is twofactor123"},
			{TestID: "submit", Role: "button", Note: "Posts the password step"},
			{TestID: "code-form", Transient: true, Note: "Appears once the password is accepted"},
			{TestID: "field-code", Role: "textbox", Transient: true, Note: "The six-digit code"},
			{TestID: "submit-code", Role: "button", Transient: true, Note: "Completes the second factor"},
			{TestID: "pending-notice", Transient: true, Note: "Says the login is half done"},
			{TestID: "welcome", Transient: true, Note: "Only once both factors are in"},
			{TestID: "send-magic-link", Role: "button", Note: "Issues a sign-in link for the user account"},
			{TestID: "magic-link", Transient: true, Note: "The link, shown here because there is no inbox to open"},
			{TestID: "login-error", Transient: true, Note: "Why the last step was refused"},
		},
		Endpoints: []challenge.Endpoint{
			{Method: http.MethodPost, Path: "/classic/two-factor", Note: "The password step"},
			{Method: http.MethodPost, Path: "/classic/two-factor/code", Note: "The second factor"},
			{Method: http.MethodPost, Path: "/classic/two-factor/magic", Note: "Issues a sign-in link"},
			{Method: http.MethodGet, Path: "/classic/two-factor/magic/{token}", Note: "Redeems one, exactly once"},
			{Method: http.MethodGet, Path: "/api/control/auth", Note: "The secret, the code valid now, and every pending link"},
		},
		Stability: challenge.Stable,
	}

	view := func(req *http.Request, message string) render.View {
		store := auth.For(session.MustFromContext(req.Context()))
		return render.View{
			Title:     meta.Title,
			Challenge: &meta,
			Data: twoFactorView{
				Login:      store.Current(),
				CSRF:       store.CSRF(),
				Error:      message,
				MagicLinks: store.PendingMagicLinks(),
			},
		}
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				renderer.Page(w, req, "classic/two-factor", view(req, ""))
			})

			r.Post("/", func(w http.ResponseWriter, req *http.Request) {
				req.ParseForm()
				sess := session.MustFromContext(req.Context())
				store := auth.For(sess)

				if req.PostFormValue("csrf") != store.CSRF() {
					renderer.PageStatus(w, req, http.StatusForbidden, "classic/two-factor",
						view(req, "the CSRF token did not match"))
					return
				}

				_, throttled, ok := store.Attempt(
					sess.Clock.Now(), req.PostFormValue("username"), req.PostFormValue("password"))
				switch {
				case throttled:
					renderer.PageStatus(w, req, http.StatusTooManyRequests, "classic/two-factor",
						view(req, "too many attempts"))
				case !ok:
					renderer.PageStatus(w, req, http.StatusOK, "classic/two-factor",
						view(req, "that username and password do not match"))
				default:
					http.Redirect(w, req, meta.URL, http.StatusSeeOther)
				}
			})

			r.Post("/code", func(w http.ResponseWriter, req *http.Request) {
				req.ParseForm()
				sess := session.MustFromContext(req.Context())
				store := auth.For(sess)

				if !store.CheckTOTP(sess.Clock.Now(), req.PostFormValue("code")) {
					renderer.PageStatus(w, req, http.StatusOK, "classic/two-factor",
						view(req, "that code is not valid at this moment"))
					return
				}
				store.CompleteSecondFactor()
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})

			r.Post("/magic", func(w http.ResponseWriter, req *http.Request) {
				auth.For(session.MustFromContext(req.Context())).IssueMagicToken("user")
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})

			r.Get("/magic/{token}", func(w http.ResponseWriter, req *http.Request) {
				sess := session.MustFromContext(req.Context())
				store := auth.For(sess)

				username, ok := store.RedeemMagicToken(chi.URLParam(req, "token"))
				if !ok {
					renderer.PageStatus(w, req, http.StatusNotFound, "classic/two-factor",
						view(req, "that link has already been used, or never existed"))
					return
				}
				if user, found := auth.Lookup(username); found {
					store.SignIn(sess.Clock.Now(), user)
				}
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})

			r.Post("/logout", func(w http.ResponseWriter, req *http.Request) {
				auth.For(session.MustFromContext(req.Context())).LogOut()
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})
		},
	}
}
