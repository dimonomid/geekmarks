// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package postgres

import (
	"database/sql"
	"fmt"
	"strconv"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"github.com/dimonomid/interrors"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) CreateBookmark(tx *sql.Tx, bd *storage.BookmarkData) (bkmID int, err error) {
	// If URL is not empty, check whether the bookmark with the same URL already exists
	if bd.URL != "" {
		existingBkms, err := s.GetBookmarksByURL(tx, bd.URL, bd.OwnerID, &storage.TagsFetchOpts{
			TagsFetchMode:     storage.TagsFetchModeNone,
			TagNamesFetchMode: storage.TagNamesFetchModeNone,
		})
		if err != nil {
			return 0, errors.Trace(err)
		}

		if len(existingBkms) > 0 {
			return 0, errors.Errorf("bookmark with the url %q already exists", bd.URL)
		}
	}

	bkmID, err = s.CreateTaggable(tx, &storage.TaggableData{
		OwnerID: bd.OwnerID,
		Type:    storage.TaggableTypeBookmark,
	})
	if err != nil {
		return 0, errors.Trace(err)
	}

	_, err = tx.Exec(
		"INSERT INTO bookmarks (id, url, title, comment) VALUES ($1, $2, $3, $4)",
		bkmID, bd.URL, bd.Title, bd.Comment,
	)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return bkmID, nil
}

func (s *StoragePostgres) UpdateBookmark(tx *sql.Tx, bd *storage.BookmarkData) (err error) {
	// If URL is not empty, check whether the bookmark with the same URL already exists
	if bd.URL != "" {
		existingBkms, err := s.GetBookmarksByURL(tx, bd.URL, bd.OwnerID, &storage.TagsFetchOpts{
			TagsFetchMode:     storage.TagsFetchModeNone,
			TagNamesFetchMode: storage.TagNamesFetchModeNone,
		})
		if err != nil {
			return errors.Trace(err)
		}

		if len(existingBkms) > 0 && existingBkms[0].ID != bd.ID {
			return errors.Errorf("bookmark with the url %q already exists", bd.URL)
		}
	}

	_, err = tx.Exec(
		"UPDATE bookmarks SET url = $1, title = $2, comment = $3 WHERE id = $4",
		bd.URL, bd.Title, bd.Comment, bd.ID,
	)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func setDefaultTagFetchOpts(tagsFetchOpts *storage.TagsFetchOpts) *storage.TagsFetchOpts {
	if tagsFetchOpts == nil {
		tagsFetchOpts = &storage.TagsFetchOpts{}
	}

	if tagsFetchOpts.TagsFetchMode == "" {
		tagsFetchOpts.TagsFetchMode = storage.TagsFetchModeDefault
	}

	if tagsFetchOpts.TagNamesFetchMode == "" {
		tagsFetchOpts.TagNamesFetchMode = storage.TagNamesFetchModeDefault
	}

	return tagsFetchOpts
}

func (s *StoragePostgres) GetTaggedBookmarks(
	tx *sql.Tx, tagIDs []int, ownerID *int, tagsFetchOpts *storage.TagsFetchOpts,
) (bookmarks []storage.BookmarkDataWTags, err error) {
	bookmarks = []storage.BookmarkDataWTags{}

	tagsFetchOpts = setDefaultTagFetchOpts(tagsFetchOpts)

	// TODO: currently, two queries are performed: first, we get the list of IDs,
	// and then, we fetch bookmarks with those IDs. We'll probably need to refactor
	// this.
	taggableIDs, err := s.GetTaggedTaggableIDs(
		tx, tagIDs, ownerID, []storage.TaggableType{storage.TaggableTypeBookmark},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(taggableIDs) > 0 {
		args := []interface{}{}
		for _, id := range taggableIDs {
			args = append(args, id)
		}

		tagsJsonFieldQuery, err := getTagsJsonFieldQuery(tagsFetchOpts, "t")
		if err != nil {
			return nil, hh.MakeInternalServerError(err)
		}

		rows, err := tx.Query(fmt.Sprintf(`
SELECT t.id, b.url, b.title, b.comment, t.owner_id,
       CAST(EXTRACT(EPOCH FROM t.created_ts) AS INTEGER),
       CAST(EXTRACT(EPOCH FROM t.updated_ts) AS INTEGER),
       %s as tagsjson
  FROM taggables t
  JOIN bookmarks b ON t.id = b.id
  WHERE t.id IN (`+getPlaceholdersString(1, len(taggableIDs))+`)
	`, tagsJsonFieldQuery), args...,
		)
		if err != nil {
			return nil, hh.MakeInternalServerError(err)
		}
		defer rows.Close()
		return rowsToBookmarks(rows, tagsFetchOpts)
	}

	return bookmarks, nil
}

func (s *StoragePostgres) GetBookmarksByURL(
	tx *sql.Tx, url string, ownerID int, tagsFetchOpts *storage.TagsFetchOpts,
) (bookmarks []storage.BookmarkDataWTags, err error) {
	bookmarks = []storage.BookmarkDataWTags{}

	tagsFetchOpts = setDefaultTagFetchOpts(tagsFetchOpts)

	tagsJsonFieldQuery, err := getTagsJsonFieldQuery(tagsFetchOpts, "t")
	if err != nil {
		return nil, hh.MakeInternalServerError(err)
	}

	rows, err := tx.Query(fmt.Sprintf(`
SELECT t.id, b.url, b.title, b.comment, t.owner_id,
       CAST(EXTRACT(EPOCH FROM t.created_ts) AS INTEGER),
       CAST(EXTRACT(EPOCH FROM t.updated_ts) AS INTEGER),
       %s as tagsjson
  FROM taggables t
  JOIN bookmarks b ON t.id = b.id
  WHERE t.owner_id = $1 AND b.url = $2
	`, tagsJsonFieldQuery), ownerID, url,
	)
	if err != nil {
		return nil, hh.MakeInternalServerError(err)
	}
	defer rows.Close()
	return rowsToBookmarks(rows, tagsFetchOpts)
}

func (s *StoragePostgres) GetBookmarkByID(
	tx *sql.Tx, bookmarkID int, tagsFetchOpts *storage.TagsFetchOpts,
) (bookmark *storage.BookmarkDataWTags, err error) {
	tagsFetchOpts = setDefaultTagFetchOpts(tagsFetchOpts)

	bkm := storage.BookmarkDataWTags{}
	var tagBriefData []byte

	tagsJsonFieldQuery, err := getTagsJsonFieldQuery(tagsFetchOpts, "t")
	if err != nil {
		return nil, hh.MakeInternalServerError(err)
	}

	err = tx.QueryRow(fmt.Sprintf(`
SELECT t.id, b.url, b.title, b.comment, t.owner_id,
       CAST(EXTRACT(EPOCH FROM t.created_ts) AS INTEGER),
       CAST(EXTRACT(EPOCH FROM t.updated_ts) AS INTEGER),
       %s as tagsjson
  FROM taggables t
  JOIN bookmarks b ON t.id = b.id
  WHERE t.id = $1
	`, tagsJsonFieldQuery), bookmarkID,
	).Scan(
		&bkm.ID, &bkm.URL, &bkm.Title, &bkm.Comment, &bkm.OwnerID,
		&bkm.CreatedAt, &bkm.UpdatedAt,
		&tagBriefData,
	)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, errors.Annotatef(
				interrors.WrapInternalError(
					err,
					storage.ErrBookmarkDoesNotExist,
				),
				"id %d", bookmarkID,
			)
		}
		// Some unexpected error
		return nil, hh.MakeInternalServerError(err)
	}

	bkm.Tags, err = parseTagBrief(tagBriefData, tagsFetchOpts)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &bkm, nil
}

func getPlaceholdersString(start, cnt int) string {
	ret := ""

	for i := start; i < (start + cnt); i++ {
		if ret != "" {
			ret += ","
		}
		ret += "$" + strconv.Itoa(i)
	}

	return ret
}

// rowsToBookmarks expects each row to contain the following fields, in this
// order:
//
// id, url, title, comment, owner_id, created_time, updated_time, tags_data.
// For some details on what is tags_data, see parseTagBrief().
func rowsToBookmarks(
	rows *sql.Rows, tagsFetchOpts *storage.TagsFetchOpts,
) (bookmarks []storage.BookmarkDataWTags, err error) {
	bookmarks = []storage.BookmarkDataWTags{}
	for rows.Next() {
		bkm := storage.BookmarkDataWTags{}
		var tagBriefData []byte
		err := rows.Scan(
			&bkm.ID, &bkm.URL, &bkm.Title, &bkm.Comment, &bkm.OwnerID,
			&bkm.CreatedAt, &bkm.UpdatedAt,
			&tagBriefData,
		)
		if err != nil {
			return nil, hh.MakeInternalServerError(err)
		}

		bkm.Tags, err = parseTagBrief(tagBriefData, tagsFetchOpts)
		if err != nil {
			return nil, errors.Trace(err)
		}

		bookmarks = append(bookmarks, bkm)
	}
	return bookmarks, nil
}
