package testutils

import (
	"database/sql"
	"strings"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
)

func PrepareTestDB(t *testing.T, si storage.Storage) error {
	// Drop all existing tables
	tables, err := getAllTables(t, si)
	if err != nil {
		return errors.Annotatef(err, "getting all table names")
	}

	t.Logf("Dropping tables: %s", strings.Join(tables, ", "))
	if len(tables) > 0 {
		err = si.Tx(func(tx *sql.Tx) error {
			_, err = tx.Exec("DROP TABLE " + strings.Join(tables, ", "))
			if err != nil {
				return errors.Annotatef(err, "dropping all tables")
			}

			return nil
		})
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Init schema (apply all migrations)
	t.Logf("Applying migrations...")
	err = si.ApplyMigrations()
	if err != nil {
		return errors.Annotatef(err, "applying migrations")
	}

	return nil
}

func CleanupTestDB(t *testing.T) error {
	// TODO: migrate down and check that no tables are present
	return nil
}

func getAllTables(t *testing.T, si storage.Storage) ([]string, error) {
	var tables []string
	err := si.Tx(func(tx *sql.Tx) error {
		rows, err := tx.Query(`
			SELECT table_name
				FROM information_schema.tables
				WHERE table_schema='public'
				AND table_type='BASE TABLE'
		`)
		if err != nil {
			return errors.Trace(err)
		}
		defer rows.Close()
		for rows.Next() {
			var tableName string
			err := rows.Scan(&tableName)
			if err != nil {
				return errors.Trace(err)
			}

			tables = append(tables, tableName)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Annotatef(err, "dropping all tables")
	}

	return tables, nil
}

func CreateTestUser(
	t *testing.T, si storage.Storage, username, password, email string,
) (userID int, err error) {
	err = si.Tx(func(tx *sql.Tx) error {
		var err error
		userID, err = si.CreateUser(tx, &storage.UserData{
			Username: username,
			Password: password,
			Email:    email,
		})
		if err != nil {
			return errors.Annotatef(
				err, "creating test user: username %q, password %q, email %q",
				username, password, email,
			)
		}
		return nil
	})
	return userID, errors.Trace(err)
}
