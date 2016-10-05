package main

import (
	"net/http"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"github.com/juju/errors"
)

type testType2 struct {
	Username *string `json:"username"`
	Email    *string `json:"email"`
}

func userTagsGet(r *http.Request, getUser GetUser) (resp interface{}, err error) {
	ud, err := getUser(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	authorized, err := authorizeOperation(r, &authzArgs{
		OwnerID: ud.ID,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	if !authorized {
		return nil, hh.MakeForbiddenError()
	}

	resp = testType2{
		Username: &ud.Username,
		Email:    &ud.Email,
	}

	return resp, nil
}
