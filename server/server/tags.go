package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"goji.io/pattern"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/storage"

	"github.com/juju/errors"
)

type userTagsGetResp struct {
	Tags []userTagData `json:"tags"`
}

type userTagData struct {
	ID          int           `json:"id"`
	Description string        `json:"description"`
	Names       []string      `json:"names"`
	Subtags     []userTagData `json:"subtags,omitempty"`
}

type userTagsPostArgs struct {
	Names       []string `json:"names"`
	Description string   `json:"description"`
}

type userTagsPostResp struct {
	TagID int `json:"tagID"`
}

func (gm *GMServer) getParentTagIDFromPath(r *http.Request, tx *sql.Tx, ownerID int) (int, error) {
	path := pattern.Path(r.Context())
	parentTagID := 0

	if len(path) > 0 {
		if parentID, err := strconv.Atoi(path[1:]); err == nil {
			parentTagData, err := gm.si.GetTag(tx, parentID, &storage.GetTagOpts{})
			if err != nil {
				return 0, errors.Trace(err)
			}
			ok, err := gm.authorizeOperation(r, &authzArgs{OwnerID: parentTagData.OwnerID})
			if err != nil {
				return 0, errors.Trace(err)
			}
			if !ok {
				return 0, hh.MakeForbiddenError()
			}
			parentTagID = parentID
		}
	}

	if parentTagID == 0 {
		var err error
		parentTagID, err = gm.si.GetTagIDByPath(tx, ownerID, path)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return parentTagID, nil
}

func (gm *GMServer) userTagsGet(
	r *http.Request, gu getUser,
) (interface{}, error) {
	ud, err := gm.getUserAndAuthorize(r, gu, &authzArgs{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	var tagData *storage.TagData
	err = gm.si.Tx(func(tx *sql.Tx) error {
		parentTagID, err := gm.getParentTagIDFromPath(r, tx, ud.ID)
		if err != nil {
			return errors.Trace(err)
		}

		tagData, err = gm.si.GetTag(tx, parentTagID, &storage.GetTagOpts{
			GetNames:   true,
			GetSubtags: true,
		})
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	resp := gm.createUserTagData(tagData)

	return resp, nil
}

func (gm *GMServer) createUserTagData(in *storage.TagData) *userTagData {
	if in == nil {
		return nil
	}

	res := userTagData{
		ID:          in.ID,
		Description: in.Description,
		Names:       in.Names,
	}

	for _, td := range in.Subtags {
		res.Subtags = append(res.Subtags, *gm.createUserTagData(&td))
	}

	return &res
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
		parentTagID, err := gm.getParentTagIDFromPath(r, tx, ud.ID)
		if err != nil {
			return errors.Trace(err)
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
