// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package postgres

import (
	"database/sql"

	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

// Implements storage.Storage
type StoragePostgres struct {
	postgresURL string
	db          *sql.DB
}

func New(postgresURL string) (*StoragePostgres, error) {
	return &StoragePostgres{
		postgresURL: postgresURL,
	}, nil
}

func (s *StoragePostgres) Connect() error {
	var err error
	s.db, err = sql.Open("postgres", s.postgresURL)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (s *StoragePostgres) ApplyMigrations() error {
	mig, err := initMigrations()
	if err != nil {
		return errors.Trace(err)
	}

	err = mig.MigrateToLatest(s.db)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
