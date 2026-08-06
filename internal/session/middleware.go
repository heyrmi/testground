package session

import "net/http"

// Header and Cookie carry the session id. The header wins when both are
// present, so a test runner can pin a worker's session even in a browser that
// is already carrying a cookie.
const (
	Header = "X-Playground-Session"
	Cookie = "playground_session"
)

// Middleware attaches an isolated session to every request and echoes its id
// back on the response, so a client that never asked for one can still learn
// which session it landed in.
func (s *Store) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested, ok := requestedID(r)
		if !ok {
			http.Error(w, "invalid session id: up to 64 characters from [A-Za-z0-9_-]", http.StatusBadRequest)
			return
		}

		sess := s.Open(ID(requested))

		w.Header().Set(Header, string(sess.ID))
		if cookie, err := r.Cookie(Cookie); err != nil || cookie.Value != string(sess.ID) {
			setCookie(w, sess.ID)
		}

		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), sess)))
	})
}

// requestedID reports the session id the client asked for, and whether the
// request was well formed. An unusable cookie is treated as no cookie rather
// than as an error, so a stale browser cannot wedge itself out of the site.
func requestedID(r *http.Request) (id string, ok bool) {
	if header := r.Header.Get(Header); header != "" {
		return header, ValidID(header)
	}
	if cookie, err := r.Cookie(Cookie); err == nil && ValidID(cookie.Value) {
		return cookie.Value, true
	}
	return "", true
}

func setCookie(w http.ResponseWriter, id ID) {
	http.SetCookie(w, &http.Cookie{
		Name:  Cookie,
		Value: string(id),
		Path:  "/",
		// Readable from JavaScript on purpose: driving the playground from
		// page scripts is a legitimate thing for a test to do.
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}
