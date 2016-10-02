package middleware

import "net/http"

type genericMiddleware struct {
	f func(w http.ResponseWriter, r *http.Request)
}

func (h *genericMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.f(w, r)
}

func MkMiddleware(f func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return &genericMiddleware{
		f: f,
	}
}
