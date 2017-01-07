package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"goji.io"
	"goji.io/pat"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/middleware"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/golang/glog"
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

func parseBearerAuth(r *http.Request) (token string, ok bool) {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	return header[len(prefix):], true
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
		token, ok := parseBearerAuth(r)

		if !ok {
			// When connecting via websocket protocok, JavaScript API does not have a
			// way to provide a bearer HTTP authorization header, so we use a trick
			// here: if there is a basic auth header with an empty password, then we
			// interpret username as a token.
			glog.V(2).Infof("Failed to parse Bearer auth, falling back to the basic auth, and interpreting the username as a token")
			var username, password string
			username, password, ok = r.BasicAuth()
			if ok {
				if username != "" && password == "" {
					glog.V(2).Infof("Interpreting username %q as a token", username)
					token = username
				} else {
					glog.V(2).Infof("Failed to use basic auth: password should be empty, username should not.")
				}
			}
		}

		if ok {
			var ud *storage.UserData
			err := gm.si.Tx(func(tx *sql.Tx) error {
				ud2, err := gm.si.GetUserByAccessToken(tx, token)

				if err != nil {
					return errors.Trace(err)
				}

				ud = ud2
				return nil
			})
			if err != nil {
				w.Header().Set("WWW-Authenticate", "Bearer realm=\"login please\"")
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

type GoogleTokenInfo struct {
	// Audience: Who is the intended audience for this token. In general the
	// same as issued_to.
	Audience string `json:"audience,omitempty"`

	// Email: The email address of the user. Present only if the email scope
	// is present in the request.
	Email string `json:"email,omitempty"`

	// ExpiresIn: The expiry time of the token, as number of seconds left
	// until expiry.
	ExpiresIn int64 `json:"expires_in,omitempty"`

	// IssuedTo: To whom was the token issued to. In general the same as
	// audience.
	IssuedTo string `json:"issued_to,omitempty"`

	// UserID: The obfuscated user id.
	UserID string `json:"user_id,omitempty"`

	// VerifiedEmail: Boolean flag which is true if the email address is
	// verified. Present only if the email scope is present in the request.
	VerifiedEmail bool `json:"verified_email,omitempty"`

	// Returned in case of error.
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// userID can be 0: in this case, if there is no record for Google user,
// a new GeekMarks user will also be created.
//
// If, however, userID > 0, then a new Google user will be associated with
// the existing GeekMarks user.
func (gm *GMServer) handleOAuthGoogle(
	tx *sql.Tx, code, redirectURL string,
	oauthCreds *OAuthCreds,
	endpoint oauth2.Endpoint,
	userID int,
) (uid int, googleTokenInfo *GoogleTokenInfo, err error) {
	if code == "" {
		return 0, nil, errors.Errorf("code is required")
	}

	if redirectURL == "" {
		return 0, nil, errors.Errorf("redirect_uri is required")
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
		return 0, nil, errors.Annotatef(err, "failed to exchange code for the token")
	}

	// Request token info by id_token

	idToken, ok := tok.Extra("id_token").(string)
	if !ok {
		return 0, nil, errors.Annotatef(err, "failed to get id_token data from the token")
	}

	qsVals := url.Values{}
	qsVals.Add("id_token", idToken)

	hc := &http.Client{Timeout: 10 * time.Second}
	tokenInfoURL := fmt.Sprintf(
		"https://www.googleapis.com/oauth2/v2/tokeninfo?%s", qsVals.Encode(),
	)
	tresp, err := hc.Get(tokenInfoURL)
	if err != nil {
		return 0, nil, errors.Annotatef(err, "could not verify token %q", idToken)
	}

	googleTokenInfo = &GoogleTokenInfo{}

	decoder := json.NewDecoder(tresp.Body)
	if err := decoder.Decode(googleTokenInfo); err != nil {
		return 0, nil, errors.Annotatef(err, "failed to decode google token info")
	}

	if tresp.StatusCode != http.StatusOK {
		return 0, nil, errors.Annotatef(err, "error getting token info: %q", *googleTokenInfo)
	}

	// Check if we have a record for that Google user
	ud, err := gm.si.GetUserByGoogleUserID(tx, googleTokenInfo.UserID)
	if err != nil {
		if errors.Cause(err) != storage.ErrUserDoesNotExist {
			// Some unexpected error
			return 0, nil, errors.Trace(err)
		}

		// We don't have a record for that Google user: let's create one
		glog.V(2).Infof("No record for the Google user %q, going to create..", googleTokenInfo.UserID)

		if userID == 0 {
			var err error
			glog.V(2).Infof("Creating a new GeekMarks user..")

			userID, err = gm.si.CreateUser(tx, &storage.UserData{
				Username: googleTokenInfo.Email,
				Email:    googleTokenInfo.Email,
			})
			if err != nil {
				return 0, nil, hh.MakeInternalServerError(err)
			}
		} else {
			glog.V(2).Infof("Using user id %d..", userID)
		}

		glog.V(2).Infof("Associating google user %q with GeekMarks user %d", googleTokenInfo.UserID, userID)
		if err := gm.si.CreateGoogleUser(
			tx, userID, googleTokenInfo.UserID, googleTokenInfo.Email,
		); err != nil {
			return 0, nil, errors.Trace(err)
		}
	} else {
		userID = ud.ID
		glog.V(2).Infof("Google user %q (email %q) belongs to user id %d",
			googleTokenInfo.UserID, googleTokenInfo.Email, userID,
		)
	}

	return userID, googleTokenInfo, nil
}

func (gm *GMServer) authenticatePostOAuthGoogle(
	tx *sql.Tx, gmr *GMRequest, oauthCreds *OAuthCreds, endpoint oauth2.Endpoint,
) (resp interface{}, err error) {
	code := gmr.FormValue("code")
	redirectURL := gmr.FormValue("redirect_uri")

	userID, googleTokenInfo, err := gm.handleOAuthGoogle(
		tx, code, redirectURL, oauthCreds, endpoint, 0,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Get or create GeekMarks access token
	tokenDescr := fmt.Sprintf(
		"Created for Google user %q (email: %q)",
		googleTokenInfo.UserID, googleTokenInfo.Email,
	)
	glog.V(2).Infof("Getting or creating geekmarks token: %q", tokenDescr)

	token, err := gm.si.GetAccessToken(tx, userID, tokenDescr, true)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return map[string]string{
		"token": token,
	}, nil
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

	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		switch provider {
		case providerGoogle:
			resp, err = gm.authenticatePostOAuthGoogle(tx, gmr, oauthCreds, googleEndpoint)
			if err != nil {
				return errors.Trace(err)
			}
		default:
			return hh.MakeInternalServerError(
				errors.Errorf("auth provider %q exists, but is not handled", provider),
			)
		}
		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return resp, nil
}

func (gm *GMServer) setupAuthAPIEndpoints(mux *goji.Mux, gsu getSubjUser) {
	setUserEndpoint(pat.Get("/client_id"), gm.oauthClientIDGet, nil, mux, gsu)
	setUserEndpoint(pat.Post("/authenticate"), gm.authenticatePost, nil, mux, gsu)
}
