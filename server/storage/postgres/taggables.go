package postgres

import (
	"database/sql"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) CreateTaggable(tx *sql.Tx, tgbd *storage.TaggableData) (tgbID int, err error) {
	err = tx.QueryRow(
		"INSERT INTO taggables (owner_id, type) VALUES ($1, $2) RETURNING id",
		tgbd.OwnerID, tgbd.Type,
	).Scan(&tgbID)
	if err != nil {
		return 0, hh.MakeInternalServerError(errors.Annotatef(
			err, "adding new taggable (owner_id: %d, type: %s)", tgbd.OwnerID, tgbd.Type,
		))
	}

	return tgbID, nil
}
