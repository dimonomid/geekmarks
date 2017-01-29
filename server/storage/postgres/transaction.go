package postgres

import (
	"database/sql"
	"fmt"

	"github.com/golang/glog"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

type TxILevel int

const (
	TxILevelReadCommitted TxILevel = iota
	TxILevelRepeatableRead
	TxILevelSerializable
)

type TxMode int

const (
	TxModeReadWrite TxMode = iota
	TxModeReadOnly
)

func (s *StoragePostgres) Tx(fn func(*sql.Tx) error) error {
	return s.TxOpt(TxILevelReadCommitted, TxModeReadWrite, fn)
}

func (s *StoragePostgres) TxOpt(
	ilevel TxILevel, mode TxMode, fn func(*sql.Tx) error,
) error {

	if ilevel != TxILevelReadCommitted && mode == TxModeReadWrite {
		// TODO: implement retrying of read-write transactions in case of
		// (RepeatableRead or Serializable) and ReadWrite, and write tests which
		// concurrently do lots of updates
		return errors.Errorf("read-write mode is currently supported only for \"Read Committed\" isolation level")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return errors.Annotate(err, "begin transaction")
	}

	// Adjust transaction params (isolation level and access mode), if needed {{{
	if ilevel != TxILevelReadCommitted {
		if _, err := tx.Exec(
			fmt.Sprintf("SET TRANSACTION ISOLATION LEVEL %s", ilevelToString(ilevel)),
		); err != nil {
			return errors.Annotate(err, "set isolation level")
		}
	}

	if mode == TxModeReadOnly {
		if _, err := tx.Exec("SET TRANSACTION READ ONLY"); err != nil {
			return errors.Annotate(err, "set isolation level")
		}
	}
	// }}}

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

func ilevelToString(ilevel TxILevel) string {
	switch ilevel {
	case TxILevelReadCommitted:
		return "READ COMMITTED"
	case TxILevelRepeatableRead:
		return "REPEATABLE READ"
	case TxILevelSerializable:
		return "SERIALIZABLE"
	}
	panic(fmt.Sprintf("unknown isolation level: %d", ilevel))
}
