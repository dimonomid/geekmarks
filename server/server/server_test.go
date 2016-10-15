// +build all_tests integration_tests

package server

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"dmitryfrank.com/geekmarks/server/interror"
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
		t.Errorf("%s", interror.ErrorStack(err))
	}

	err = si.Connect()
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	gminstance, err := New(si)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	err = testutils.PrepareTestDB(t, si)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	handler, err := gminstance.CreateHandler()
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	ts := httptest.NewServer(handler)
	defer ts.Close()

	err = f(si, ts)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	err = testutils.CleanupTestDB(t)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
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

// Ignores IDs
func tagDataEqual(tdExpected, tdGot *userTagData) error {
	if tdExpected.Description != tdGot.Description {
		return errors.Errorf("expected tag descr %q, got %q", tdExpected.Description, tdGot.Description)
	}

	if !reflect.DeepEqual(tdExpected.Names, tdGot.Names) {
		return errors.Errorf("expected names %v, got %v", tdExpected.Names, tdGot.Names)
	}

	if len(tdExpected.Subtags) != len(tdGot.Subtags) {
		return errors.Errorf("expected subtags len %d, got %d", len(tdExpected.Subtags), len(tdGot.Subtags))
	}

	for k, _ := range tdExpected.Subtags {
		if err := tagDataEqual(&tdExpected.Subtags[k], &tdGot.Subtags[k]); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func doReq(method, url, username, password string, body io.Reader, checkHTTPCode bool) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if checkHTTPCode {
		if err := expectHTTPCode(resp, http.StatusOK); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return resp, nil
}

func addTag(url, username, password, names, descr string) (int, error) {
	resp, err := doReq(
		"POST", url, username, password,
		bytes.NewReader([]byte(fmt.Sprintf(`{"names": [%s], "description": "%s"}`, names, descr))),
		true,
	)
	if err != nil {
		return 0, errors.Trace(err)
	}

	var respMap map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&respMap)

	tagID, ok := respMap["tagID"]
	if !ok {
		return 0, errors.Errorf("response %v does not contain tagID", respMap)
	}
	if tagID.(float64) <= 0 {
		return 0, errors.Errorf("tagID should be > 0, but got %d", tagID)
	}
	return int(tagID.(float64)), nil
}

func TestTagsGetSet(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, ts *httptest.Server) error {
		var err error

		if _, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}

		if _, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}

		var tagID_Foo1, tagID_Foo3, tagID_Foo1_a, tagID_Foo1_b, tagID_Foo1_b_c int

		// Get initial tag tree (should be only root tag)
		{
			resp, err := doReq(
				"GET", fmt.Sprintf("%s/api/my/tags", ts.URL), "test1", "1", nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			var tdGot userTagData
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&tdGot)

			tdExpected := userTagData{
				Names:       []string{""},
				Description: "Root pseudo-tag",
				Subtags:     []userTagData{},
			}

			err = tagDataEqual(&tdExpected, &tdGot)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// Try to add tag foo1 (foo2)
		tagID_Foo1, err = addTag(
			fmt.Sprintf("%s/api/my/tags", ts.URL), "test1", "1", `"foo1", "foo2"`, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag which already exists (should fail)
		{
			resp, err := doReq(
				"POST", fmt.Sprintf("%s/api/my/tags", ts.URL), "test1", "1",
				bytes.NewReader([]byte(`
				{"names": ["foo3", "foo2", "foo4"]}
				`)),
				false,
			)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(
				resp, http.StatusBadRequest, "Tag with the name \"foo2\" already exists",
			); err != nil {
				return errors.Trace(err)
			}
		}

		// Try to add tag foo3
		tagID_Foo3, err = addTag(
			fmt.Sprintf("%s/api/my/tags", ts.URL), "test1", "1", `"foo3"`, "my foo 3 tag",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo1 / a
		tagID_Foo1_a, err = addTag(
			fmt.Sprintf("%s/api/my/tags/foo1", ts.URL), "test1", "1", `"a"`, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo2 / b (note that foo1 is the same as foo2)
		tagID_Foo1_b, err = addTag(
			fmt.Sprintf("%s/api/my/tags/foo2", ts.URL), "test1", "1", `"b"`, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo2 / b / c, specifying parent as ID, not path
		tagID_Foo1_b_c, err = addTag(
			fmt.Sprintf("%s/api/my/tags/%d", ts.URL, tagID_Foo1_b), "test1", "1", `"c"`, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Get resulting tag tree
		{
			resp, err := doReq(
				"GET", fmt.Sprintf("%s/api/my/tags", ts.URL), "test1", "1", nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			var tdGot userTagData
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&tdGot)

			tdExpected := userTagData{
				Names:       []string{""},
				Description: "Root pseudo-tag",
				Subtags: []userTagData{
					userTagData{
						Names:       []string{"foo1", "foo2"},
						Description: "",
						Subtags: []userTagData{
							userTagData{
								Names:       []string{"a"},
								Description: "",
								Subtags:     []userTagData{},
							},
							userTagData{
								Names:       []string{"b"},
								Description: "",
								Subtags: []userTagData{
									userTagData{
										Names:       []string{"c"},
										Description: "",
										Subtags:     []userTagData{},
									},
								},
							},
						},
					},
					userTagData{
						Names:       []string{"foo3"},
						Description: "my foo 3 tag",
						Subtags:     []userTagData{},
					},
				},
			}

			err = tagDataEqual(&tdExpected, &tdGot)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// Get resulting tag tree from tag foo1 / b
		{
			resp, err := doReq(
				"GET", fmt.Sprintf("%s/api/my/tags/foo1/b", ts.URL), "test1", "1", nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			resp2, err := doReq(
				"GET", fmt.Sprintf("%s/api/my/tags/%d", ts.URL, tagID_Foo1_b), "test1", "1", nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			var tdGot userTagData
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&tdGot)

			var tdGot2 userTagData
			decoder = json.NewDecoder(resp2.Body)
			decoder.Decode(&tdGot2)

			tdExpected := userTagData{
				Names:       []string{"b"},
				Description: "",
				Subtags: []userTagData{
					userTagData{
						Names:       []string{"c"},
						Description: "",
						Subtags:     []userTagData{},
					},
				},
			}

			err = tagDataEqual(&tdExpected, &tdGot)
			if err != nil {
				return errors.Trace(err)
			}

			err = tagDataEqual(&tdExpected, &tdGot2)
			if err != nil {
				return errors.Trace(err)
			}
		}

		fmt.Println(tagID_Foo1, tagID_Foo3, tagID_Foo1_a, tagID_Foo1_b, tagID_Foo1_b_c)

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
