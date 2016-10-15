package server

import (
	"database/sql"
	"encoding/json"
	"strconv"

	"dmitryfrank.com/geekmarks/server/interror"
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

func (gm *GMServer) getTagIDFromPath(gmr *GMRequest, tx *sql.Tx, ownerID int) (int, error) {
	parentTagID := 0

	if len(gmr.Path) > 0 {
		if parentID, err := strconv.Atoi(gmr.Path[1:]); err == nil {
			parentTagData, err := gm.si.GetTag(tx, parentID, &storage.GetTagOpts{})
			if err != nil {
				return 0, errors.Trace(err)
			}
			err = gm.authorizeOperation(
				gmr.Caller, &authzArgs{OwnerID: parentTagData.OwnerID},
			)
			if err != nil {
				return 0, errors.Trace(err)
			}
			parentTagID = parentID
		}
	}

	if parentTagID == 0 {
		var err error
		parentTagID, err = gm.si.GetTagIDByPath(tx, ownerID, gmr.Path)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return parentTagID, nil
}

func (gm *GMServer) userTagsGet(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	var tagData *storage.TagData
	err = gm.si.Tx(func(tx *sql.Tx) error {
		parentTagID, err := gm.getTagIDFromPath(gmr, tx, gmr.SubjUser.ID)
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

	resp = gm.createUserTagData(tagData)

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

func (gm *GMServer) userTagsPost(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	decoder := json.NewDecoder(gmr.Body)
	var args userTagsPostArgs
	err = decoder.Decode(&args)
	if err != nil {
		// TODO: provide request data example
		return nil, interror.WrapInternalError(
			err,
			errors.Errorf("invalid data"),
		)
	}

	tagID := 0

	err = gm.si.Tx(func(tx *sql.Tx) error {
		parentTagID, err := gm.getTagIDFromPath(gmr, tx, gmr.SubjUser.ID)
		if err != nil {
			return errors.Trace(err)
		}

		tagID, err = gm.si.CreateTag(tx, &storage.TagData{
			OwnerID:     gmr.SubjUser.ID,
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
