package postgres

import (
	"database/sql"

	"github.com/golang/glog"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) Tx(fn func(*sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return errors.Annotate(err, "begin transaction")
	}

	err = fn(tx)
	if err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			glog.Errorf("Transaction rollback failed: %+v", err2)
		}
		return errors.Trace(err)
	}

	err = tx.Commit()
	if err != nil {
		return errors.Annotate(err, "commit transaction")
	}
	return nil
}
