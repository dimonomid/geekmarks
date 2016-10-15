// +build all_tests integration_tests

package server

import (
	"encoding/json"
	"flag"
	"fmt"
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

func TestTagsGet(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, ts *httptest.Server) error {
		var u1ID, u2ID int
		var err error

		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}

		if u2ID, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}

		var u1TagsGetRespByPath, u1TagsGetRespByMy []byte
		var u2TagsGetRespByPath, u2TagsGetRespByMy []byte

		// test1 requests its own tags
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/users/%d/tags", ts.URL, u1ID), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.SetBasicAuth("test1", "1")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u1TagsGetRespByPath, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// test1 requests its own tags via /api/my
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/my/tags", ts.URL), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.SetBasicAuth("test1", "1")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u1TagsGetRespByMy, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// test1 requests FOREIGN tags, should fail
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/users/%d/tags", ts.URL, u2ID), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.SetBasicAuth("test1", "1")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(resp, http.StatusForbidden, "forbidden"); err != nil {
				return errors.Trace(err)
			}
		}

		// test2 requests its own tags
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/users/%d/tags", ts.URL, u2ID), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.SetBasicAuth("test2", "2")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u2TagsGetRespByPath, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// test2 requests its own tags via /api/my
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/my/tags", ts.URL), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.SetBasicAuth("test2", "2")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u2TagsGetRespByMy, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// check that responses match and mismatch as expected

		if string(u1TagsGetRespByPath) != string(u1TagsGetRespByMy) {
			return errors.Errorf("u1TagsGetRespByPath should be equal to u1TagsGetRespByMy")
		}

		if string(u2TagsGetRespByPath) != string(u2TagsGetRespByMy) {
			return errors.Errorf("u2TagsGetRespByPath should be equal to u2TagsGetRespByMy")
		}

		if string(u1TagsGetRespByPath) == string(u2TagsGetRespByPath) {
			return errors.Errorf("u1TagsGetRespByPath should NOT be equal to u2TagsGetRespByPath")
		}

		return nil
	})
}

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
