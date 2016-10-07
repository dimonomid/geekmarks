// +build all_tests integration_tests

package postgres

import (
	"flag"
	"os"
	"strings"
	"testing"

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
		t.Errorf("%s", err)
	}

	err = si.Connect()
	if err != nil {
		t.Errorf("%s", err)
	}

	err = testutils.PrepareTestDB(t, si)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = f(si)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = testutils.CleanupTestDB(t)
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestTagWithInvalidOwner(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		_, err := si.db.Exec(
			"INSERT INTO tags (parent_id, owner_id) VALUES (NULL, 100)",
		)
		if err == nil {
			return errors.Errorf("should be error")
		}
		if !strings.Contains(err.Error(), "foreign") {
			return errors.Errorf("error should contain \"foreign\", but it doesn't: %q", err)
		}
		return nil
	})
}

func TestTagWithNullOwner(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		_, err := si.db.Exec(
			"INSERT INTO tags (parent_id, owner_id) VALUES (NULL, NULL)",
		)
		if err == nil {
			return errors.Errorf("should be error")
		}
		if !strings.Contains(err.Error(), "not-null") {
			return errors.Errorf("error should contain \"not-null\", but it doesn't: %q", err)
		}
		return nil
	})
}
