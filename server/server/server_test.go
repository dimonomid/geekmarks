// +build all_tests integration_tests

package server

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	storagecommon "dmitryfrank.com/geekmarks/server/storage/common"
	"dmitryfrank.com/geekmarks/server/testutils"
	"github.com/juju/errors"
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

func runWithRealDB(t *testing.T, f func(si storage.Storage, ts *httptest.Server) error) {
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

	err = f(si, ts)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = testutils.CleanupTestDB(t)
	if err != nil {
		t.Errorf("%s", err)
	}
}

func TestInternalError(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, ts *httptest.Server) error {
		resp, err := http.Get(ts.URL + "/api/test_internal_error")
		if err != nil {
			return errors.Trace(err)
		}

		if resp.StatusCode != http.StatusInternalServerError {
			return errors.Errorf(
				"HTTP Status Code: expected %d, got %d",
				http.StatusInternalServerError, resp.StatusCode,
			)
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Trace(err)
		}

		t.Log("=====body====")
		t.Log(string(body))
		t.Log("=====body end====")

		return nil
	})
}

func TestUnauthorized(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, ts *httptest.Server) error {
		var err error

		resp, err := http.Get(ts.URL + "/api/my/tags")
		if err != nil {
			return errors.Trace(err)
		}

		if err := expectErrorResp(resp, http.StatusUnauthorized, "unauthorized"); err != nil {
			return errors.Trace(err)
		}

		// Any URL under "my" should return 401
		resp, err = http.Get(ts.URL + "/api/my/foo/bar/baz")
		if err != nil {
			return errors.Trace(err)
		}

		if err := expectErrorResp(resp, http.StatusUnauthorized, "unauthorized"); err != nil {
			return errors.Trace(err)
		}

		return nil
	})
}

//TODO

//func TestForbidden(t *testing.T) {
//runWithRealDB(t, func(si storage.Storage, ts *httptest.Server) error {
////var u1ID, u2ID int
//var err error

////if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
////return errors.Trace(err)
////}

////if u2ID, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
////return errors.Trace(err)
////}

//resp, err := http.Get(ts.URL + "/api/my/tags")
//if err != nil {
//return errors.Trace(err)
//}

//if err := expectHTTPCode(resp, http.StatusUnauthorized); err != nil {
//return errors.Trace(err)
//}

//return nil
//})
//}

func expectHTTPCode(resp *http.Response, code int) error {
	if resp.StatusCode != code {
		return errors.Errorf(
			"HTTP Status Code: expected %d, got %d",
			code, resp.StatusCode,
		)
	}
	return nil
}

func expectErrorResp(resp *http.Response, code int, message string) error {
	if err := expectHTTPCode(resp, code); err != nil {
		return errors.Trace(err)
	}

	rmap, err := getRespMap(resp)
	if err != nil {
		return errors.Trace(err)
	}

	exp := map[string]interface{}{
		"status":  float64(code),
		"message": message,
	}
	if !reflect.DeepEqual(exp, rmap) {
		return errors.Errorf("response JSON: expected: %v, got: %v", exp, rmap)
	}

	return nil
}

func getRespMap(resp *http.Response) (map[string]interface{}, error) {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	v := map[string]interface{}{}

	err = json.Unmarshal(body, &v)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return v, nil
}
