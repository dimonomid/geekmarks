package postgres

//go:generate go-bindata -nocompress -modtime 1 -mode 420 -pkg postgres migrations

import (
	"database/sql"

	"github.com/golang/glog"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
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
	migrations := &migrate.AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      "migrations",
	}

	n, err := migrate.Exec(s.db, "postgres", migrations, migrate.Up)
	if n == 0 {
		glog.Infof("No migrations applied")
	} else {
		glog.Infof("Applied %d migrations!", n)
	}
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
