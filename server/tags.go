package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"dmitryfrank.com/geekmarks/server/storage"

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

type userTagsPostArgs struct {
	Path  *string  `json:"path,omitempty"`
	Names []string `json:"names"`
}

type userTagsPostResp struct {
	TagID int `json:"tagID"`
}

func userTagsPost(r *http.Request, getUser GetUser) (resp interface{}, err error) {
	ud, err := getUserAndAuthorize(r, getUser, &authzArgs{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	decoder := json.NewDecoder(r.Body)
	var args userTagsPostArgs
	err = decoder.Decode(&args)
	if err != nil {
		// TODO: provide request data example
		return nil, errors.Errorf("invalid data")
	}

	tagID := 0

	err = storage.Tx(func(tx *sql.Tx) error {
		var err error

		parentTagID := 0
		// If parent tag ID is provided, use it; otherwise, get the root tag id for
		// the user
		if args.Path != nil {
			parentTagID, err = storage.GetTagIDByPath(tx, ud.ID, *args.Path)
			if err != nil {
				return errors.Trace(err)
			}
		} else {
			parentTagID, err = storage.GetRootTagID(tx, ud.ID)
			if err != nil {
				return errors.Trace(err)
			}
		}

		tagID, err = storage.CreateTag(tx, ud.ID, parentTagID, args.Names)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	resp = userTagsPostResp{
		TagID: tagID,
	}

	return resp, nil
}
