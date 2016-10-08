package server

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

func (gm *GMServer) userTagsGet(r *http.Request, gu getUser) (resp interface{}, err error) {
	ud, err := gm.getUserAndAuthorize(r, gu, &authzArgs{})
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
	ParentPath  *string  `json:"parentPath,omitempty"`
	ParentID    *int     `json:"parentID,omitempty"`
	Names       []string `json:"names"`
	Description string   `json:"description"`
}

type userTagsPostResp struct {
	TagID int `json:"tagID"`
}

func (gm *GMServer) userTagsPost(r *http.Request, gu getUser) (resp interface{}, err error) {
	ud, err := gm.getUserAndAuthorize(r, gu, &authzArgs{})
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

	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error

		parentTagID := 0
		// If parent tag ID is provided, use it; otherwise, get the root tag id for
		// the user
		if args.ParentPath != nil {
			parentTagID, err = gm.si.GetTagIDByPath(tx, ud.ID, *args.ParentPath)
			if err != nil {
				return errors.Trace(err)
			}
		} else if args.ParentID != nil {
			ownerID, err := gm.si.GetTagOwnerByID(tx, *args.ParentID)
			if err != nil {
				return errors.Trace(err)
			}
			ok, err := gm.authorizeOperation(r, &authzArgs{OwnerID: ownerID})
			if err != nil {
				return errors.Trace(err)
			}
			if !ok {
				return hh.MakeForbiddenError()
			}
			parentTagID = *args.ParentID
		} else {
			parentTagID, err = gm.si.GetRootTagID(tx, ud.ID)
			if err != nil {
				return errors.Trace(err)
			}
		}

		tagID, err = gm.si.CreateTag(tx, &storage.TagData{
			OwnerID:     ud.ID,
			ParentTagID: parentTagID,
			Names:       args.Names,
			Description: args.Description,
		})
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
