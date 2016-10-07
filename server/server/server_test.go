package server

import (
	"database/sql"
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	storagecommon "dmitryfrank.com/geekmarks/server/storage/common"
	"github.com/juju/errors"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func runWithRealDB(t *testing.T, f func(ts *httptest.Server) error) {
	si, err := storagecommon.CreateStorage()
	if err != nil {
		t.Errorf("%s", err)
	}

	err = si.Connect()
	if err != nil {
		t.Errorf("%s", err)
	}

	gminstance, err := New(si)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = dbPrepare(t, si)
	if err != nil {
		t.Errorf("%s", err)
	}

	handler, err := gminstance.CreateHandler()
	if err != nil {
		t.Errorf("%s", err)
	}

	ts := httptest.NewServer(handler)
	defer ts.Close()

	err = f(ts)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = dbCleanup(t)
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestInternalError(t *testing.T) {
	runWithRealDB(t, func(ts *httptest.Server) error {
		res, err := http.Get(ts.URL + "/api/test_internal_error")
		if err != nil {
			return errors.Trace(err)
		}

		if res.StatusCode != http.StatusInternalServerError {
			return errors.Errorf(
				"HTTP Status Code: expected %d, got %d",
				http.StatusInternalServerError, res.StatusCode,
			)
		}

		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return errors.Trace(err)
		}

		t.Log("=====body====")
		t.Log(string(body))
		t.Log("=====body end====")

		return nil
	})
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

func dbPrepare(t *testing.T, si storage.Storage) error {
	// Drop all existing tables
	tables, err := getAllTables(t, si)
	if err != nil {
		return errors.Annotatef(err, "getting all table names")
	}

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
	err = si.ApplyMigrations()
	if err != nil {
		return errors.Annotatef(err, "applying migrations")
	}

	return nil
}

func dbCleanup(t *testing.T) error {
	// TODO: migrate down and check that no tables are present
	return nil
}
