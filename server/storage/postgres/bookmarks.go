package postgres

import (
	"database/sql"

	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) CreateBookmark(tx *sql.Tx, bd *storage.BookmarkData) (bkmID int, err error) {
	bkmID, err = s.CreateTaggable(tx, &storage.TaggableData{
		OwnerID: bd.OwnerID,
		Type:    storage.TaggableTypeBookmark,
	})
	if err != nil {
		return 0, errors.Trace(err)
	}

	_, err = tx.Exec(
		"INSERT INTO bookmarks (id, url, comment) VALUES ($1, $2, $3)",
		bkmID, bd.URL, bd.Comment,
	)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return bkmID, nil
}
