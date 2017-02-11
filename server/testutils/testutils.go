// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

// +build all_tests unit_tests integration_tests

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

	enums, err := getAllEnums(t, si)
	if err != nil {
		return errors.Annotatef(err, "getting all enum names")
	}

	t.Logf("Dropping enums: %s", strings.Join(enums, ", "))
	if len(enums) > 0 {
		err = si.Tx(func(tx *sql.Tx) error {
			_, err = tx.Exec("DROP TYPE " + strings.Join(enums, ", "))
			if err != nil {
				return errors.Annotatef(err, "dropping all enums")
			}

			return nil
		})
		if err != nil {
			return errors.Trace(err)
		}
	}

	// Drop other custom types
	// TODO: find a universal way to drop all user-defined types

	// We ignore errors on purpose: if the type doesn't exist, we just don't care
	err = si.Tx(func(tx *sql.Tx) error {
		tx.Exec("DROP TYPE gm_tag_brief")

		return nil
	})
	if err != nil {
		return errors.Trace(err)
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
		return nil, errors.Annotatef(err, "getting all tables")
	}

	return tables, nil
}

func getAllEnums(t *testing.T, si storage.Storage) ([]string, error) {
	var types []string
	err := si.Tx(func(tx *sql.Tx) error {
		rows, err := tx.Query(`
			SELECT typname FROM pg_type t
			JOIN pg_enum e ON t.oid = e.enumtypid
			GROUP BY typname;
		`)
		if err != nil {
			return errors.Trace(err)
		}
		defer rows.Close()
		for rows.Next() {
			var typeName string
			err := rows.Scan(&typeName)
			if err != nil {
				return errors.Trace(err)
			}

			types = append(types, typeName)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Annotatef(err, "getting all enum types")
	}

	return types, nil
}

func CreateTestUser(
	si storage.Storage, username, email string,
) (userID int, token string, err error) {
	err = si.Tx(func(tx *sql.Tx) error {
		var err error
		userID, err = si.CreateUser(tx, &storage.UserData{
			Username: username,
			Email:    email,
		})
		if err != nil {
			return errors.Annotatef(
				err, "creating test user: username %q, email %q", username, email,
			)
		}

		token, err = si.GetAccessToken(tx, userID, "test token for a test user", true)
		if err != nil {
			return errors.Annotatef(
				err, "creating test access token for the user id %d", userID,
			)
		}

		return nil
	})
	return userID, token, errors.Trace(err)
}

func DeleteTestUser(
	si storage.Storage, userID int,
) error {
	err := si.Tx(func(tx *sql.Tx) error {
		if err := si.DeleteUser(tx, userID); err != nil {
			return errors.Annotatef(err, "deleting test user %d", userID)
		}

		return nil
	})
	return errors.Trace(err)
}
