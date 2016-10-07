package server

import (
	"context"
	"database/sql"
	"net/http"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/middleware"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
)

func (gm *GMServer) authnRequiredMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value("authUserData")
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
func (gm *GMServer) authnMiddleware(inner http.Handler) http.Handler {
	mw := func(w http.ResponseWriter, r *http.Request) {
		// TODO: use https://github.com/abbot/go-http-auth for digest auth
		username, password, ok := r.BasicAuth()
		if ok {

			var ud *storage.UserData
			err := gm.si.Tx(func(tx *sql.Tx) error {
				ud2, err := gm.si.GetUser(tx, &storage.GetUserArgs{
					Username: cptr.String(username),
				})

				if err != nil {
					if errors.Cause(err) == sql.ErrNoRows {
						// User does not exist
						return hh.MakeUnauthorizedError()
					}

					// Some unexpected error
					return hh.MakeInternalServerError(errors.Annotatef(err, "checking auth"))
				}

				if ud2.Password != password {
					// User exists, but the password is wrong
					return hh.MakeUnauthorizedError()
				}

				ud = ud2
				return nil
			})
			if err != nil {
				w.Header().Set("WWW-Authenticate", "Basic realm=\"login please\"")
				hh.RespondWithError(w, r, err)
				return
			}

			// Authn data is correct: create a new request with updated context
			ctx := r.Context()
			ctx = context.WithValue(ctx, "authUserData", ud)
			r = r.WithContext(ctx)
		}

		// Process request, whether authn data was not provided at all, or was
		// provided correctly.
		inner.ServeHTTP(w, r)
	}
	return middleware.MkMiddleware(mw)
}
