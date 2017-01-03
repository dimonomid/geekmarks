package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	"goji.io"
	"goji.io/pat"

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

func getAuthnUserDataByReq(r *http.Request) *storage.UserData {
	v := r.Context().Value("authUserData")
	if v == nil {
		// Not authenticated
		return nil
	}

	return v.(*storage.UserData)
}

func (gm *GMServer) oauthClientIDGet(gmr *GMRequest) (resp interface{}, err error) {
	provider := pat.Param(gmr.HttpReq, "provider")
	oauthCreds, ok := gm.oauthProviders[provider]
	if !ok {
		return nil, errors.Errorf("unknown auth provider: %q", provider)
	}

	resp = clientIDGetResp{
		ClientID: oauthCreds.ClientID,
	}

	return resp, nil
}

func (gm *GMServer) authenticatePostOAuth(
	gmr *GMRequest, oauthCreds *OAuthCreds, endpoint oauth2.Endpoint,
) (resp interface{}, err error) {
	code := gmr.FormValue("code")
	redirectURL := gmr.FormValue("redirect_uri")

	if code == "" {
		return nil, errors.Errorf("code is required")
	}

	if redirectURL == "" {
		return nil, errors.Errorf("redirect_uri is required")
	}

	ctx := context.Background()
	conf := &oauth2.Config{
		ClientID:     oauthCreds.ClientID,
		ClientSecret: oauthCreds.ClientSecret,
		Scopes:       []string{"email"},
		Endpoint:     endpoint,
		RedirectURL:  redirectURL,
	}

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return nil, errors.Annotatef(err, "failed to exchange code for the token")
	}

	fmt.Println("token!", tok)

	return map[string]string{
		// TODO: real token
		"token": "foobar",
	}, nil

	// TODO: get path to the credentials file from the command line flag,
	//       and read client id and secret from there
	//
	// TODO: add google_auth table:
	// google_user_id string
	// email string
	// user_id integer

	// TODO: add auth_tokens table:
	// user_id integer
	// token string

	// TODO: here, get token info, like this (you may see
	// cloud/common/google_auth.go):
	//
	// https://www.googleapis.com/oauth2/v2/tokeninfo?id_token=foobar
	// (where foobar is tok.raw["id_token"], NOTE: tok.raw needs to be
	// casted to map[string]string), you'll get something like:
	// {
	//   "issued_to": "1061779365092-n2tv8e96h20q2l5l52u8v9q1b6hi52kc.apps.googleusercontent.com",
	//   "audience": "1061779365092-n2tv8e96h20q2l5l52u8v9q1b6hi52kc.apps.googleusercontent.com",
	//   "user_id": "112797053189661838135",
	//   "expires_in": 3504,
	//   "email": "dubinin2004@gmail.com",
	//   "verified_email": true
	// }

	//
	// then check google_auth table for the item with the "google_user_id"
	// equal to "user_id" from the above, and if it exists, then get our
	// user_id from it, and return a corresponding token (from the
	// auth_tokens table) or create a new row, and also create a new user
	// and a token.
	//
	// In either case, we should return our geekmarks token to the caller.
}

func (gm *GMServer) authenticatePost(gmr *GMRequest) (resp interface{}, err error) {
	provider := pat.Param(gmr.HttpReq, "provider")
	oauthCreds, ok := gm.oauthProviders[provider]
	if !ok {
		return nil, errors.Errorf("unknown auth provider: %q", provider)
	}

	if oauthCreds == nil {
		return nil, errors.Errorf("auth provider %q is disabled (corresponding flag to the creds file was not provided)", provider)
	}

	switch provider {
	case providerGoogle:
		return gm.authenticatePostOAuth(gmr, oauthCreds, googleEndpoint)
	default:
		return nil, hh.MakeInternalServerError(
			errors.Errorf("auth provider %q exists, but is not handled", provider),
		)
	}
}

func (gm *GMServer) setupAuthAPIEndpoints(mux *goji.Mux, gsu getSubjUser) {
	setUserEndpoint(pat.Get("/client_id"), gm.oauthClientIDGet, nil, mux, gsu)
	setUserEndpoint(pat.Post("/authenticate"), gm.authenticatePost, nil, mux, gsu)
}
