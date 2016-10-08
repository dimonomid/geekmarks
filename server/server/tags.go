package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

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

type userTagsParent struct {
	ParentPath *string `json:"parentPath,omitempty"`
	ParentID   *int    `json:"parentID,omitempty"`
}

type userTagsPostArgs struct {
	userTagsParent `json:",omitempty"`
	Names          []string `json:"names"`
	Description    string   `json:"description"`
}

type userTagsPostResp struct {
	TagID int `json:"tagID"`
}

func (up *userTagsParent) getParentID(
	r *http.Request, tx *sql.Tx, userID int, gm *GMServer,
) (int, error) {
	var err error
	parentTagID := 0
	// If parent tag ID is provided, use it; otherwise, get the root tag id for
	// the user
	if up.ParentPath != nil {
		parentTagID, err = gm.si.GetTagIDByPath(tx, userID, *up.ParentPath)
		if err != nil {
			return 0, errors.Trace(err)
		}
	} else if up.ParentID != nil {
		parentTagData, err := gm.si.GetTag(tx, *up.ParentID, &storage.GetTagOpts{})
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
		parentTagID = *up.ParentID
	} else {
		parentTagID, err = gm.si.GetRootTagID(tx, userID)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return parentTagID, nil
}

func (gm *GMServer) userTagsGet(
	r *http.Request, gu getUser,
) (resp interface{}, err error) {
	ud, err := gm.getUserAndAuthorize(r, gu, &authzArgs{})
	if err != nil {
		return nil, errors.Trace(err)
	}

	up := userTagsParent{}
	if parentIdStr := r.FormValue("parent_id"); parentIdStr != "" {
		parentID, err := strconv.Atoi(parentIdStr)
		if err != nil {
			return nil, errors.Trace(err)
		}
		up.ParentID = &parentID
	}
	if parentPathStr := r.FormValue("parent_id"); parentPathStr != "" {
		up.ParentPath = &parentPathStr
	}

	var tagsData []storage.TagData
	err = gm.si.Tx(func(tx *sql.Tx) error {
		parentTagID, err := up.getParentID(r, tx, ud.ID, gm)
		if err != nil {
			return errors.Trace(err)
		}

		tagsData, err = gm.si.GetTags(tx, parentTagID, &storage.GetTagOpts{
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

	resp = userTagsGetResp{
		Tags: gm.createUserTagsData(tagsData),
	}

	return resp, nil
}

func (gm *GMServer) createUserTagsData(in []storage.TagData) []userTagData {
	if in == nil {
		return nil
	}

	res := []userTagData{}

	for _, td := range in {
		resTD := userTagData{
			ID:          td.ID,
			Description: td.Description,
			Names:       td.Names,
			Subtags:     gm.createUserTagsData(td.Subtags),
		}
		res = append(res, resTD)
	}

	return res
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
		parentTagID, err := args.getParentID(r, tx, ud.ID, gm)
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
