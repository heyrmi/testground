package classic

import (
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"

	"github.com/heyrmi/testground/internal/challenge"
	"github.com/heyrmi/testground/internal/httpx"
	"github.com/heyrmi/testground/internal/render"
	"github.com/heyrmi/testground/internal/session"
)

// submissions holds what a session last posted to one challenge.
type submissions[T any] struct {
	mu     sync.Mutex
	latest *T
	count  int
}

func (s *submissions[T]) record(value T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = &value
	s.count++
}

func (s *submissions[T]) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.latest = nil
	s.count = 0
}

func (s *submissions[T]) read() (*T, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest, s.count
}

// formView is what a Classic form template renders from.
type formView[T any] struct {
	// Latest is nil until something has been posted, which is how a template
	// tells "no submission yet" from "submitted an empty form".
	Latest *T
	Count  int
}

// formPage is the shape nearly every Classic challenge takes: a GET that
// renders the form beside whatever the last submission produced, and a POST
// that records one and answers 303 so the browser re-fetches the page. The
// redirect is not incidental -- it means a refresh cannot resubmit, and it
// means every element reference a test was holding is now stale.
func formPage[T any](meta challenge.Challenge, read func(*http.Request) T) page {
	state := func(r *http.Request) *submissions[T] {
		return session.Value(session.MustFromContext(r.Context()), meta.ID, func() *submissions[T] {
			return &submissions[T]{}
		})
	}

	return page{
		meta: meta,
		mount: func(r chi.Router, renderer *render.Renderer) {
			template := "classic/" + meta.ID

			r.Get("/", func(w http.ResponseWriter, req *http.Request) {
				latest, count := state(req).read()
				renderer.Page(w, req, template, render.View{
					Title:     meta.Title,
					Challenge: &meta,
					Data:      formView[T]{Latest: latest, Count: count},
				})
			})

			r.Post("/", func(w http.ResponseWriter, req *http.Request) {
				if err := req.ParseForm(); err != nil {
					httpx.Fail(w, http.StatusBadRequest, "could not parse the form")
					return
				}
				state(req).record(read(req))
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})

			r.Post("/clear", func(w http.ResponseWriter, req *http.Request) {
				state(req).clear()
				http.Redirect(w, req, meta.URL, http.StatusSeeOther)
			})
		},
	}
}
