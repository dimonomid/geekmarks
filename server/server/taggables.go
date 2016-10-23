package server

import (
	"database/sql"
	"encoding/json"
	"strconv"

	"dmitryfrank.com/geekmarks/server/cptr"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"

	"github.com/juju/errors"
)

const (
	TagID = "tag_id"
)

type userBookmarkData struct {
	ID        int    `json:"id"`
	URL       string `json:"url"`
	Comment   string `json:"comment,omitempty"`
	UpdatedAt uint64 `json:"updatedAt"`
}

type userBookmarkPostArgs struct {
	URL     string `json:"url"`
	Comment string `json:"comment,omitempty"`
	TagIDs  []int  `json:"tagIDs"`
}

type userBookmarkPostResp struct {
	BookmarkID int `json:"bookmarkID"`
}

func (gm *GMServer) userBookmarksGet(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	tagIDs := []int{}
	for _, stid := range gmr.Values[TagID] {
		v, err := strconv.Atoi(stid)
		if err != nil {
			return nil, errors.Annotatef(err, "wrong tag id %q", stid)
		}
		tagIDs = append(tagIDs, v)
	}

	var bkms []storage.BookmarkData

	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		bkms, err = gm.si.GetTaggedBookmarks(tx, tagIDs, cptr.Int(gmr.SubjUser.ID))
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	bkmsUser := []userBookmarkData{}

	for _, bkm := range bkms {
		bkmsUser = append(bkmsUser, userBookmarkData{
			ID:        bkm.ID,
			URL:       bkm.URL,
			Comment:   bkm.Comment,
			UpdatedAt: bkm.UpdatedAt,
		})
	}

	return bkmsUser, nil
}

func (gm *GMServer) userBookmarksPost(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	decoder := json.NewDecoder(gmr.Body)
	var args userBookmarkPostArgs
	err = decoder.Decode(&args)
	if err != nil {
		// TODO: provide request data example
		return nil, interror.WrapInternalError(
			err,
			errors.Errorf("invalid data"),
		)
	}

	bkmID := 0

	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		bkmID, err = gm.si.CreateBookmark(tx, &storage.BookmarkData{
			OwnerID: gmr.SubjUser.ID,
			Comment: args.Comment,
			URL:     args.URL,
		})
		if err != nil {
			return errors.Trace(err)
		}

		err = gm.si.SetTaggings(
			tx, bkmID, args.TagIDs, storage.TaggingModeLeafs,
		)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	resp = userBookmarkPostResp{
		BookmarkID: bkmID,
	}
	return resp, nil
}
