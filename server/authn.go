package main

import (
	"context"
	"net/http"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/middleware"
)

func authnRequiredMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value("username")
		if v == nil {
			// No authentication data: respond with an error
			w.Header().Set("WWW-Authenticate", "Basic realm=\"login please\"")
			hh.RespondWithError(w, r, hh.MakeUnauthorizedError())
			return
		}

		// Authentication data is found; proceed.
		inner.ServeHTTP(w, r)
	}
	return middleware.MkMiddleware(mw)
}

// Middleware which populates the context with the authentication data, if
// it is provided and is correct.
//
// If it's provided but isn't correct, responds with an error. TODO: do we
// really need that behaviour? Maybe it's better to just proceed without authn
// data? Dunno.
//
// NOTE: be sure to use it after httphelper.MakeDesiredContentTypeMiddleware(),
// since the error response should be in the right format
func authnMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		// TODO: use https://github.com/abbot/go-http-auth for digest auth
		username, password, ok := r.BasicAuth()
		if ok {
			if !(username == "alice" && password == "alice") &&
				!(username == "bob" && password == "bob") {
				// Authn data is provided but is wrong: respond with an error
				w.Header().Set("WWW-Authenticate", "Basic realm=\"login please\"")
				hh.RespondWithError(w, r, hh.MakeUnauthorizedError())
				return
			}

			// Authn data is correct: create a new request with updated context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "username", username)
			r = r.WithContext(ctx)
		}

		// Process request, whether authn data was not provided at all, or was
		// provided correctly.
		inner.ServeHTTP(w, r)
	}
	return middleware.MkMiddleware(mw)
}
