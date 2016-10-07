// +build all_tests integration_tests

package server

import (
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	storagecommon "dmitryfrank.com/geekmarks/server/storage/common"
	"dmitryfrank.com/geekmarks/server/testutils"
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

	err = testutils.PrepareTestDB(t, si)
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

	err = testutils.CleanupTestDB(t)
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
