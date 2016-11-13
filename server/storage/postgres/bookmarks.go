package postgres

import (
	"database/sql"
	"fmt"
	"strconv"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) CreateBookmark(tx *sql.Tx, bd *storage.BookmarkData) (bkmID int, err error) {
	if bd.URL == "" {
		return 0, errors.Errorf("url should not be empty")
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

func (s *StoragePostgres) GetTaggedBookmarks(
	tx *sql.Tx, tagIDs []int, ownerID *int, tagsFetchOpts *storage.TagsFetchOpts,
) (bookmarks []storage.BookmarkDataWTags, err error) {
	bookmarks = []storage.BookmarkDataWTags{}

	if tagsFetchOpts == nil {
		tagsFetchOpts = &storage.TagsFetchOpts{}
	}

	if tagsFetchOpts.TagsFetchMode == "" {
		tagsFetchOpts.TagsFetchMode = storage.TagsFetchModeDefault
	}

	if tagsFetchOpts.TagNamesFetchMode == "" {
		tagsFetchOpts.TagNamesFetchMode = storage.TagNamesFetchModeDefault
	}

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
	}

	return bookmarks, nil
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
