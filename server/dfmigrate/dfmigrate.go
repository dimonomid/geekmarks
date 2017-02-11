// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package dfmigrate

import (
	"database/sql"

	"github.com/golang/glog"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

type MigrationFunc func(tx *sql.Tx) error

type Migration struct {
	id    int
	descr string
	up    MigrationFunc
	down  MigrationFunc
}

type Migrations struct {
	migrations []Migration
}

func (m *Migrations) AddMigration(
	id int, descr string, up MigrationFunc, down MigrationFunc,
) error {
	if id != len(m.migrations)+1 {
		return errors.Errorf("wrong migration id for %q: expected %d, given %d",
			descr, len(m.migrations)+1, id,
		)
	}
	m.migrations = append(m.migrations, Migration{
		id, descr, up, down,
	})
	return nil
}

func (m *Migrations) MigrateToLatest(db *sql.DB) error {
	return m.Migrate(db, len(m.migrations))
}

func (m *Migrations) Migrate(db *sql.DB, targetMigrationID int) error {
	err := initialize(db)
	if err != nil {
		return errors.Trace(err)
	}

	if targetMigrationID > len(m.migrations) {
		return errors.Errorf("wrong target migration id %d (max: %d)",
			targetMigrationID, len(m.migrations),
		)
	}

	curID, err := getCurrentMigrationID(db)
	if err != nil {
		return errors.Trace(err)
	}

	if curID > len(m.migrations) {
		return errors.Errorf("wrong saved current migration id %d (max: %d)",
			curID, len(m.migrations),
		)
	}

	if curID < 0 {
		return errors.Errorf("wrong saved current migration id %d", curID)
	}

	if targetMigrationID > curID {
		// migrate up
		for _, mig := range m.migrations[curID:] {
			glog.Infof("Applying migration %d %q", mig.id, mig.descr)
			err := tx(db, mig.up)
			if err != nil {
				return errors.Trace(err)
			}

			err = setCurrentMigrationID(db, mig.id)
			if err != nil {
				return errors.Trace(err)
			}
			glog.Infof("Applied successfully")
		}
	} else if targetMigrationID < curID {
		// migrate down
		// TODO
		return errors.Errorf("migration down is not implemented")
	}

	return nil
}
