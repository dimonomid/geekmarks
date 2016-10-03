package main

import (
	"context"
	"net/http"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/middleware"
)

func authMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		// TODO: use https://github.com/abbot/go-http-auth for digest auth
		username, password, ok := r.BasicAuth()
		if !ok {
			hh.RespondWithError(w, hh.MakeUnauthorizedError())
			return
		}

		if !(username == "alice" && password == "alice") &&
			!(username == "bob" && password == "bob") {
			hh.RespondWithError(w, hh.MakeUnauthorizedError())
			return
		}

		// Process request
		inner.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "username", username)))
	}
	return middleware.MkMiddleware(mw)
}
