package main

import (
	"net/http"

	"github.com/juju/errors"
)

type testType2 struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
}

func userTagsGet(r *http.Request, getUser GetUser) (resp interface{}, err error) {
	ud, err := getUserAndAuthorize(r, getUser, &authzArgs{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	resp = testType2{
		Username: &ud.Username,
		Email:    &ud.Email,
	}

	return resp, nil
}
