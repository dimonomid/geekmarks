// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

// +build all_tests integration_tests

package postgres

import (
	"database/sql"
	"flag"
	"os"
	"testing"

	"dmitryfrank.com/geekmarks/server/cptr"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/testutils"

	"github.com/juju/errors"
)

var (
	postgresURL = flag.String("geekmarks.postgres.url", "",
		"Data source name pointing to the Postgres database. Alternatively, can be "+
			"given in an environment variable GM_POSTGRES_URL.")
)

func runWithRealDB(t *testing.T, f func(si *StoragePostgres) error) {
	pgURL := *postgresURL
	if pgURL == "" {
		pgURL = os.Getenv("GM_POSTGRES_URL")
	}
	si, err := New(pgURL)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
		return
	}

	err = si.Connect()
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
		return
	}

	err = testutils.PrepareTestDB(t, si)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
		return
	}

	err = f(si)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
		return
	}

	err = testutils.CleanupTestDB(t)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
		return
	}
}

func TestTransactionRollback(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		var u1ID int
		var err error
		if u1ID, _, err = testutils.CreateTestUser(si, "test1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}

		var rootTagID int
		err = si.Tx(func(tx *sql.Tx) error {
			var err error
			rootTagID, err = si.GetRootTagID(tx, u1ID)
			if err != nil {
				return errors.Annotatef(err, "getting root tag for user %d", u1ID)
			}

			_, err = si.CreateTag(tx, &storage.TagData{
				OwnerID:     u1ID,
				ParentTagID: cptr.Int(rootTagID),
				Description: cptr.String("test tag2"),
				Names:       []string{"normal_name", "123"},
			})
			return errors.Trace(err)
		})
		if err == nil || errors.Cause(err) != storage.ErrTagNameInvalid {
			return errors.Errorf("should not be able to create tag with the name 123")
		}

		err = si.Tx(func(tx *sql.Tx) error {
			var cnt int
			err := tx.QueryRow(
				"SELECT COUNT(name) FROM tag_names WHERE name = $1", "normal_name",
			).Scan(&cnt)
			if err != nil {
				return errors.Annotatef(err, "getting count of tag names")
			}
			if cnt > 0 {
				return errors.Errorf("there should be 0 tag names, but there is %d", cnt)
			}

			return nil
		})
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
}
