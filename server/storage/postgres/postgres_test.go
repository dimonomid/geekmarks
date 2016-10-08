// +build all_tests integration_tests

package postgres

import (
	"flag"
	"os"
	"testing"

	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/testutils"
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
	}

	err = si.Connect()
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	err = testutils.PrepareTestDB(t, si)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	err = f(si)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	err = testutils.CleanupTestDB(t)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}
}
