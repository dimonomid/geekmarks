package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
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
	ParentPath *string  `json:"parentPath,omitempty"`
	ParentID   *int     `json:"parentID,omitempty"`
	Names      []string `json:"names"`
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
		if args.ParentPath != nil {
			parentTagID, err = storage.GetTagIDByPath(tx, ud.ID, *args.ParentPath)
			if err != nil {
				return errors.Trace(err)
			}
		} else if args.ParentID != nil {
			ownerID, err := storage.GetTagOwnerByID(tx, *args.ParentID)
			if err != nil {
				return errors.Trace(err)
			}
			ok, err := authorizeOperation(r, &authzArgs{OwnerID: ownerID})
			if err != nil {
				return errors.Trace(err)
			}
			if !ok {
				return hh.MakeForbiddenError()
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
