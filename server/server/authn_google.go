// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/golang/glog"
	"github.com/juju/errors"
)

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
