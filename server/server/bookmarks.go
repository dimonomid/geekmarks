// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package server

import (
	"database/sql"
	"encoding/json"
	"strconv"

	"goji.io/pat"

	"dmitryfrank.com/geekmarks/server/cptr"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"

	"github.com/juju/errors"
)

const (
	QSArgBkmGetArgTagID = "tag_id"
	QSArgBkmGetArgURL   = "url"
)

type userBookmarkTag struct {
	Items []userBookmarkTagItem `json:"items"`
}

type userBookmarkTagItem struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

type userBookmarkData struct {
	ID        int               `json:"id"`
	URL       string            `json:"url"`
	Title     string            `json:"title,omitempty"`
	Comment   string            `json:"comment,omitempty"`
	UpdatedAt uint64            `json:"updatedAt"`
	Tags      []userBookmarkTag `json:"tags,omitempty"`
}

type userBookmarkPostArgs struct {
	URL     string `json:"url"`
	Title   string `json:"title,omitempty"`
	Comment string `json:"comment,omitempty"`
	TagIDs  []int  `json:"tagIDs"`
}

type userBookmarkPostResp struct {
	BookmarkID int `json:"bookmarkID"`
}

type userBookmarkPutResp struct {
}

type userBookmarkDeleteResp struct {
}

func (gm *GMServer) userBookmarksGet(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Check if both tag_id and url are given (it's an error)
	if len(gmr.Values[QSArgBkmGetArgTagID]) > 0 && len(gmr.Values[QSArgBkmGetArgURL]) > 0 {
		return nil, errors.Errorf(
			"%q and %q cannot be given both", QSArgBkmGetArgTagID, QSArgBkmGetArgURL,
		)
	}

	tagsFetchOpts := storage.TagsFetchOpts{
		TagsFetchMode:     storage.TagsFetchModeLeafs,
		TagNamesFetchMode: storage.TagNamesFetchModeFull,
	}

	var bkms []storage.BookmarkDataWTags

	if len(gmr.Values[QSArgBkmGetArgURL]) > 0 {
		// get bookmarks by URL

		err = gm.si.Tx(func(tx *sql.Tx) error {
			var err error
			bkms, err = gm.si.GetBookmarksByURL(
				tx, gmr.Values[QSArgBkmGetArgURL][0], gmr.SubjUser.ID, &tagsFetchOpts,
			)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
	} else {
		// get tagged bookmarks

		tagIDs := []int{}
		for _, stid := range gmr.Values[QSArgBkmGetArgTagID] {
			v, err := strconv.Atoi(stid)
			if err != nil {
				return nil, errors.Annotatef(err, "wrong tag id %q", stid)
			}
			tagIDs = append(tagIDs, v)
		}

		err = gm.si.Tx(func(tx *sql.Tx) error {
			var err error
			bkms, err = gm.si.GetTaggedBookmarks(
				tx, tagIDs, cptr.Int(gmr.SubjUser.ID), &tagsFetchOpts,
			)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		})
		if err != nil {
			return nil, errors.Trace(err)
		}
	}

	bkmsUser := []userBookmarkData{}

	for _, bkm := range bkms {
		bkmsUser = append(bkmsUser, userBookmarkData{
			ID:        bkm.ID,
			URL:       bkm.URL,
			Title:     bkm.Title,
			Comment:   bkm.Comment,
			UpdatedAt: bkm.UpdatedAt,
			Tags:      getUserBookmarkTags(bkm.Tags),
		})
	}

	return bkmsUser, nil
}

func (gm *GMServer) userBookmarkGet(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	bkmID, err := getBookmarkIDFromQueryString(gmr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var bkm *storage.BookmarkDataWTags

	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		bkm, err = gm.si.GetBookmarkByID(
			tx, bkmID, &storage.TagsFetchOpts{
				TagsFetchMode:     storage.TagsFetchModeLeafs,
				TagNamesFetchMode: storage.TagNamesFetchModeFull,
			},
		)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	bkmUser := userBookmarkData{
		ID:        bkm.ID,
		URL:       bkm.URL,
		Title:     bkm.Title,
		Comment:   bkm.Comment,
		UpdatedAt: bkm.UpdatedAt,
		Tags:      getUserBookmarkTags(bkm.Tags),
	}

	return bkmUser, nil
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
			Title:   args.Title,
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

func (gm *GMServer) userBookmarkPut(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	bkmID, err := getBookmarkIDFromQueryString(gmr)
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

	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		err = gm.si.UpdateBookmark(tx, &storage.BookmarkData{
			ID:      bkmID,
			Title:   args.Title,
			Comment: args.Comment,
			URL:     args.URL,
			// NOTE: we need to pass OwnerID since it's used to check whether this
			// owner already has the bookmark with the same URL
			OwnerID: gmr.SubjUser.ID,
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

	resp = userBookmarkPutResp{}
	return resp, nil
}

func (gm *GMServer) userBookmarkDelete(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	bkmID, err := getBookmarkIDFromQueryString(gmr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	err = gm.si.Tx(func(tx *sql.Tx) error {
		if err := gm.si.DeleteTaggable(tx, bkmID); err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	resp = userBookmarkDeleteResp{}
	return resp, nil
}

func getBookmarkIDFromQueryString(gmr *GMRequest) (int, error) {
	bkmIDStr := pat.Param(gmr.HttpReq, BookmarkID)
	bkmID, err := strconv.Atoi(bkmIDStr)
	if err != nil {
		return 0, interror.WrapInternalError(
			err,
			errors.Errorf("wrong bookmark id %q", bkmIDStr),
		)
	}
	return bkmID, nil
}

// getUserBookmarkTags converts a slice of storage.BookmarkTagPath to to a
// slice of server's userBookmarkTag.
func getUserBookmarkTags(srcTags []storage.BookmarkTagPath) []userBookmarkTag {
	tags := []userBookmarkTag{}
	// TODO: Factor this out in a separate function, and use here and in
	// userBookmarksGet below
	for _, t := range srcTags {
		var items []userBookmarkTagItem
		for _, item := range t.TagItems[ /*skip root tag*/ 1:] {
			items = append(items, userBookmarkTagItem{
				ID:   item.ID,
				Name: item.Name,
			})
		}
		tags = append(tags, userBookmarkTag{
			Items: items,
		})
	}

	return tags
}
