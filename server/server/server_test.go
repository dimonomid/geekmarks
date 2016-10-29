// +build all_tests integration_tests

package server

import (
	"bytes"
	"context"
	"encoding/base64"
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
	"time"

	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"
	storagecommon "dmitryfrank.com/geekmarks/server/storage/common"
	"dmitryfrank.com/geekmarks/server/testutils"
	"github.com/gorilla/websocket"
	"github.com/juju/errors"
)

type H map[string]interface{}
type A []interface{}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

type testBackend interface {
	DoReq(
		method, url, username, password string, body io.Reader, checkHTTPCode bool,
	) (*genericResp, error)
	DoUserReq(
		method, url string, userID int, body interface{}, checkHTTPCode bool,
	) (*genericResp, error)
	GetTestServer() *httptest.Server
	SetTestServer(ts *httptest.Server)

	UserCreated(id int, username, password string)
	Close()
}

type userCreds struct {
	username string
	password string
}

type wsReq struct {
	Method string                 `json:"method"`
	Path   string                 `json:"path"`
	Values map[string]interface{} `json:"values"`
	Body   interface{}            `json:"body,omitempty"`
}

type wsResp struct {
	Method string                 `json:"method"`
	Path   string                 `json:"path"`
	Values map[string]interface{} `json:"values,omitempty"`
	Status int                    `json:"status"`
	Body   interface{}            `json:"body"`
}

type wsConn struct {
	cancel   context.CancelFunc
	stopChan chan<- struct{}

	tx chan<- wsReq
	rx <-chan wsResp
}

type testBackendOpts struct {
	UseUsersEndpoint bool
	UseWS            bool
}

type testBackendHTTP struct {
	t       *testing.T
	ts      *httptest.Server
	opts    testBackendOpts
	users   map[int]userCreds
	wsConns map[int]wsConn
}

type genericResp struct {
	StatusCode int
	Body       io.Reader
}

func makeGenericRespFromHTTPResp(resp *http.Response) (*genericResp, error) {
	return &genericResp{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
	}, nil
}

func makeGenericRespFromWSResp(resp *wsResp) (*genericResp, error) {
	data, err := json.Marshal(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &genericResp{
		StatusCode: resp.Status,
		Body:       bytes.NewReader(data),
	}, nil
}

func makeTestBackendHTTP(t *testing.T, opts testBackendOpts) *testBackendHTTP {
	return &testBackendHTTP{
		t:       t,
		users:   make(map[int]userCreds),
		wsConns: make(map[int]wsConn),
		opts:    opts,
	}
}

func (be *testBackendHTTP) DoReq(
	method, url, username, password string, body io.Reader, checkHTTPCode bool,
) (*genericResp, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", be.ts.URL, url), body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Trace(err)
	}

	genResp, err := makeGenericRespFromHTTPResp(resp)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if checkHTTPCode {
		if err := expectHTTPCode(genResp, http.StatusOK); err != nil {
			return nil, errors.Trace(err)
		}
	}

	return genResp, nil
}

func (be *testBackendHTTP) DoUserReq(
	method, url string, userID int, body interface{}, checkHTTPCode bool,
) (*genericResp, error) {
	if !be.opts.UseWS {
		creds, ok := be.users[userID]
		if !ok {
			return nil, errors.Errorf("testBackend does not have userID %d registered", userID)
		}

		fullURL := fmt.Sprintf("%s/api/my%s", be.ts.URL, url)
		if be.opts.UseUsersEndpoint {
			fullURL = fmt.Sprintf("%s/api/users/%d%s", be.ts.URL, userID, url)
		}

		data, err := json.Marshal(body)
		if err != nil {
			return nil, errors.Trace(err)
		}

		req, err := http.NewRequest(method, fullURL, bytes.NewReader(data))
		if err != nil {
			return nil, errors.Trace(err)
		}
		req.SetBasicAuth(creds.username, creds.password)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, errors.Trace(err)
		}

		genResp, err := makeGenericRespFromHTTPResp(resp)
		if err != nil {
			return nil, errors.Trace(err)
		}

		if checkHTTPCode {
			if err := expectHTTPCode(genResp, http.StatusOK); err != nil {
				return nil, errors.Trace(err)
			}
		}

		return genResp, nil
	} else {
		wsConn, ok := be.wsConns[userID]
		if !ok {
			return nil, errors.Errorf("testBackend does not have userID %d registered", userID)
		}

		//TODO: parse url, set values properly
		wsReq := wsReq{
			Method: method,
			Path:   url,
			Values: make(map[string]interface{}), //TODO
			Body:   body,
		}

		wsConn.tx <- wsReq
		wsResp := <-wsConn.rx

		if wsResp.Method != wsReq.Method {
			be.t.Errorf("ws: req method was: %q, but resp method is: %q",
				wsReq.Method, wsResp.Method,
			)
		}

		if wsResp.Path != wsReq.Path {
			be.t.Errorf("ws: req path was: %q, but resp path is: %q",
				wsReq.Path, wsResp.Path,
			)
		}

		genResp, err := makeGenericRespFromWSResp(&wsResp)
		if err != nil {
			return nil, errors.Trace(err)
		}

		if checkHTTPCode {
			if err := expectHTTPCode(genResp, http.StatusOK); err != nil {
				return nil, errors.Trace(err)
			}
		}

		return genResp, nil
	}
}

func (be *testBackendHTTP) GetTestServer() *httptest.Server {
	return be.ts
}

func (be *testBackendHTTP) SetTestServer(ts *httptest.Server) {
	be.ts = ts
}

func (be *testBackendHTTP) UserCreated(userID int, username, password string) {
	be.users[userID] = userCreds{
		username: username,
		password: password,
	}

	if be.opts.UseWS {
		h := http.Header{}
		h.Set("Authorization", "Basic "+basicAuth(username, password))

		url := "ws" + be.ts.URL[4:]
		fullURL := fmt.Sprintf("%s/api/my/wsconnect", url)
		if be.opts.UseUsersEndpoint {
			fullURL = fmt.Sprintf("%s/api/users/%d/wsconnect", url, userID)
		}
		conn, _, err := websocket.DefaultDialer.Dial(fullURL, h)
		if err != nil {
			be.t.Errorf("dial: %q %q", fullURL, err)
			return
		}

		ctx := context.Background()
		ctx, cancelFunc := context.WithTimeout(ctx, 120*time.Second)

		txChan := make(chan wsReq, 1)
		rxChan := make(chan wsResp, 1)
		stopChan := make(chan struct{}, 1)

		wsConn := wsConn{
			cancel:   cancelFunc,
			stopChan: stopChan,
			tx:       txChan,
			rx:       rxChan,
		}

		go func() {
			for {
				var wsReq wsReq

				select {
				case wsReq = <-txChan:
					w, err := conn.NextWriter(websocket.TextMessage)
					if err != nil {
						be.t.Errorf("getting ws writer: %q", errors.Trace(err))
						return
					}

					encoder := json.NewEncoder(w)
					err = encoder.Encode(wsReq)
					if err != nil {
						be.t.Errorf("encoding ws req: %q", errors.Trace(err))
						return
					}

					if err := w.Close(); err != nil {
						be.t.Errorf("closing ws writer: %q", errors.Trace(err))
						return
					}

				case <-stopChan:
					err := conn.Close()
					if err != nil {
						be.t.Errorf("closing ws: %s", errors.Trace(err))
						return
					}
					return
				case <-ctx.Done():
					conn.Close()
					be.t.Errorf("ctx.Done() is closed: %s", ctx.Err())
					return
				}

				_, reader, err := conn.NextReader()
				if err != nil {
					be.t.Errorf("getting ws reader: %s", ctx.Err())
					return
				}

				var wsResp wsResp
				decoder := json.NewDecoder(reader)
				err = decoder.Decode(&wsResp)
				if err != nil {
					be.t.Errorf("decoding ws resp: %s", errors.Trace(err))
					return
				}

				rxChan <- wsResp
			}
		}()

		be.wsConns[userID] = wsConn
	}
}

func (be *testBackendHTTP) Close() {
	for k, v := range be.wsConns {
		v.stopChan <- struct{}{}
		delete(be.wsConns, k)
	}
}

func runWithRealDB(
	t *testing.T,
	f func(si storage.Storage, be testBackend) error,
) {
	t.Logf("====== running with WebSocket, /api/my ======")
	{
		be := makeTestBackendHTTP(t, testBackendOpts{
			UseWS: true,
		})

		runWithRealDBAndBackend(t, be, f)
	}

	t.Logf("====== running with WebSocket, /api/users/X ======")
	{
		be := makeTestBackendHTTP(t, testBackendOpts{
			UseWS:            true,
			UseUsersEndpoint: true,
		})

		runWithRealDBAndBackend(t, be, f)
	}

	t.Logf("====== running with HTTP, /api/my ======")
	{
		be := makeTestBackendHTTP(t, testBackendOpts{})

		runWithRealDBAndBackend(t, be, f)
	}

	t.Logf("====== running with HTTP, /api/users/X ======")
	{
		be := makeTestBackendHTTP(t, testBackendOpts{
			UseUsersEndpoint: true,
		})

		runWithRealDBAndBackend(t, be, f)
	}
}

func runWithRealDBAndBackend(
	t *testing.T,
	be testBackend,
	f func(si storage.Storage, be testBackend) error,
) {
	defer be.Close()

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

	be.SetTestServer(ts)

	err = f(si, be)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}

	err = testutils.CleanupTestDB(t)
	if err != nil {
		t.Errorf("%s", interror.ErrorStack(err))
	}
}

func TestUnauthorized(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		ts := be.GetTestServer()
		var err error

		resp, err := http.Get(ts.URL + "/api/my/tags")
		if err != nil {
			return errors.Trace(err)
		}

		genResp, err := makeGenericRespFromHTTPResp(resp)
		if err != nil {
			return errors.Trace(err)
		}

		if err := expectErrorResp(genResp, http.StatusUnauthorized, "unauthorized"); err != nil {
			return errors.Trace(err)
		}

		// Any URL under "my" should return 401
		resp, err = http.Get(ts.URL + "/api/my/foo/bar/baz")
		if err != nil {
			return errors.Trace(err)
		}

		genResp, err = makeGenericRespFromHTTPResp(resp)
		if err != nil {
			return errors.Trace(err)
		}

		if err := expectErrorResp(genResp, http.StatusUnauthorized, "unauthorized"); err != nil {
			return errors.Trace(err)
		}

		return nil
	})
}

func TestTagsGet(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		ts := be.GetTestServer()
		var u1ID, u2ID int
		var err error

		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", "1")

		if u2ID, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u2ID, "test2", "2")

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

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
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

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
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

			genResp, err := makeGenericRespFromHTTPResp(resp)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(genResp, http.StatusForbidden, "forbidden"); err != nil {
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

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
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

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
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

func addTag(be testBackend, url string, userID int, names []string, descr string) (int, error) {
	resp, err := be.DoUserReq(
		"POST", url, userID,
		H{"names": names, "description": descr},
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

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestTagsGetSet(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		var u1ID, u2ID int
		var err error

		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", "1")

		if u2ID, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u2ID, "test2", "2")

		var tagID_Foo1, tagID_Foo3, tagID_Foo1_a, tagID_Foo1_b, tagID_Foo1_b_c int

		// Get initial tag tree (should be only root tag)
		{
			resp, err := be.DoUserReq("GET", "/tags", u1ID, nil, true)
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
			be, "/tags", u1ID, []string{"foo1", "foo2"}, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag which already exists (should fail)
		{
			resp, err := be.DoUserReq(
				"POST", "/tags", u1ID,
				H{"names": A{"foo3", "foo2", "foo4"}},
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

		// Try to add tag for another user (should fail)
		{
			resp, err := be.DoReq(
				"POST", fmt.Sprintf("/api/users/%d/tags", u2ID), "test1", "1",
				bytes.NewReader([]byte(`
				{"names": ["test"]}
				`)),
				false,
			)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(
				resp, http.StatusForbidden, "forbidden",
			); err != nil {
				return errors.Trace(err)
			}
		}

		// Try to add tag foo3
		tagID_Foo3, err = addTag(
			be, "/tags", u1ID, []string{"foo3"}, "my foo 3 tag",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo1 / a
		tagID_Foo1_a, err = addTag(
			be, "/tags/foo1", u1ID, []string{"a"}, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo2 / b (note that foo1 is the same as foo2)
		tagID_Foo1_b, err = addTag(
			be, "/tags/foo2", u1ID, []string{"b"}, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo2 / b / Привет, specifying parent as ID, not path
		tagID_Foo1_b_c, err = addTag(
			be, fmt.Sprintf("/tags/%d", tagID_Foo1_b), u1ID, []string{"Привет"}, "",
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Get resulting tag tree
		{
			resp, err := be.DoUserReq(
				"GET", "/tags", u1ID, nil, true,
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
										Names:       []string{"Привет"},
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
			resp, err := be.DoUserReq(
				"GET", "/tags/foo1/b", u1ID, nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			resp2, err := be.DoUserReq(
				"GET", fmt.Sprintf("/tags/%d", tagID_Foo1_b), u1ID, nil, true,
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
						Names:       []string{"Привет"},
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

func expectHTTPCode(resp *genericResp, code int) error {
	if resp.StatusCode != code {
		return errors.Errorf(
			"HTTP Status Code: expected %d, got %d",
			code, resp.StatusCode,
		)
	}
	return nil
}

func expectHTTPCode2(resp *http.Response, code int) error {
	if resp.StatusCode != code {
		return errors.Errorf(
			"HTTP Status Code: expected %d, got %d",
			code, resp.StatusCode,
		)
	}
	return nil
}

func expectErrorResp(resp *genericResp, code int, message string) error {
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

func getRespMap(resp *genericResp) (map[string]interface{}, error) {
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
