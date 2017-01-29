// +build all_tests integration_tests

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sync"
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
		method, url, token string, body io.Reader, checkHTTPCode bool,
	) (*genericResp, error)
	DoUserReq(
		method, url string, userID int, body interface{}, checkHTTPCode bool,
	) (*genericResp, error)
	GetTestServer() *httptest.Server
	SetTestServer(ts *httptest.Server)

	UserCreated(id int, username, token string)
	DeleteUser(id int) error
	Close()
}

type userCreds struct {
	token string
}

type wsReq struct {
	Id     int                    `json:"id"`
	Method string                 `json:"method"`
	Path   string                 `json:"path"`
	Values map[string]interface{} `json:"values"`
	Body   interface{}            `json:"body,omitempty"`
}

type wsResp struct {
	Id     int                    `json:"id"`
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
	t    *testing.T
	ts   *httptest.Server
	opts testBackendOpts

	users    map[int]userCreds
	usersMtx sync.Mutex

	wsConns    map[int]wsConn
	wsConnsMtx sync.Mutex
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
	method, url, token string, body io.Reader, checkHTTPCode bool,
) (*genericResp, error) {
	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", be.ts.URL, url), body)
	if err != nil {
		return nil, errors.Trace(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

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
	method, rawURL string, userID int, body interface{}, checkHTTPCode bool,
) (*genericResp, error) {
	if !be.opts.UseWS {
		be.usersMtx.Lock()
		creds, ok := be.users[userID]
		be.usersMtx.Unlock()
		if !ok {
			return nil, errors.Errorf("testBackend does not have userID %d registered", userID)
		}

		fullURL := fmt.Sprintf("%s/api/my%s", be.ts.URL, rawURL)
		if be.opts.UseUsersEndpoint {
			fullURL = fmt.Sprintf("%s/api/users/%d%s", be.ts.URL, userID, rawURL)
		}

		data := []byte{}
		var err error
		if body != nil {
			data, err = json.Marshal(body)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}

		req, err := http.NewRequest(method, fullURL, bytes.NewReader(data))
		if err != nil {
			return nil, errors.Trace(err)
		}
		req.Header.Set("Authorization", "Bearer "+creds.token)

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
		be.wsConnsMtx.Lock()
		wsConn, ok := be.wsConns[userID]
		be.wsConnsMtx.Unlock()
		if !ok {
			return nil, errors.Errorf("testBackend does not have userID %d registered", userID)
		}

		pURL, err := url.Parse(rawURL)
		if err != nil {
			return nil, errors.Trace(err)
		}

		values, err := url.ParseQuery(pURL.RawQuery)
		if err != nil {
			return nil, errors.Trace(err)
		}

		values2 := make(map[string]interface{})
		for k, v := range values {
			values2[k] = v
		}

		id := rand.Int()

		wsReq := wsReq{
			Id:     id,
			Method: method,
			Path:   pURL.Path,
			Values: values2,
			Body:   body,
		}

		wsConn.tx <- wsReq
		wsResp := <-wsConn.rx

		if wsResp.Id != wsReq.Id {
			be.t.Errorf("ws: req id was: %d, but resp id is: %d",
				wsReq.Id, wsResp.Id,
			)
		}

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

func (be *testBackendHTTP) DeleteUser(userID int) error {
	_, err := be.DoUserReq(
		"DELETE", "/test_user_delete", userID,
		H{}, true,
	)
	if err != nil {
		return errors.Trace(err)
	}

	be.usersMtx.Lock()
	defer be.usersMtx.Unlock()
	if _, ok := be.users[userID]; !ok {
		return errors.Errorf("testBackend does not have userID %d registered", userID)
	}
	delete(be.users, userID)

	if be.opts.UseWS {
		be.wsConnsMtx.Lock()
		defer be.wsConnsMtx.Unlock()
		wsConn, ok := be.wsConns[userID]
		if !ok {
			return errors.Errorf("testBackend does not have userID %d registered", userID)
		}
		wsConn.cancel()
		delete(be.wsConns, userID)
	}

	return nil
}

func (be *testBackendHTTP) UserCreated(userID int, username, token string) {
	be.usersMtx.Lock()
	be.users[userID] = userCreds{
		token: token,
	}
	be.usersMtx.Unlock()

	if be.opts.UseWS {
		h := http.Header{}
		h.Set("Authorization", "Bearer "+token)

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
						//be.t.Errorf("closing ws: %s", errors.Trace(err))
						return
					}
					return
				case <-ctx.Done():
					conn.Close()
					//be.t.Errorf("ctx.Done() is closed: %s", ctx.Err())
					return
				}

				_, reader, err := conn.NextReader()
				if err != nil {
					be.t.Errorf("getting ws reader: %s", ctx.Err())
					return
				}

				var wsResp wsResp
				decoder := json.NewDecoder(reader)
				decoder.UseNumber()
				err = decoder.Decode(&wsResp)
				if err != nil {
					be.t.Errorf("decoding ws resp: %s", errors.Trace(err))
					return
				}

				rxChan <- wsResp
			}
		}()

		be.wsConnsMtx.Lock()
		be.wsConns[userID] = wsConn
		be.wsConnsMtx.Unlock()
	}
}

func (be *testBackendHTTP) Close() {
	be.wsConnsMtx.Lock()
	defer be.wsConnsMtx.Unlock()
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

	// Before running tests, check database integrity, just in case (for all users)
	err = si.CheckIntegrity()
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

	// After test function ran, check database integrity (for all users)
	err = si.CheckIntegrity()
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

func expectHTTPCode(resp *genericResp, code int) error {
	if resp.StatusCode != code {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Trace(err)
		}

		return errors.Errorf(
			"HTTP Status Code: expected %d, got %d (body: %q)",
			code, resp.StatusCode, body,
		)
	}
	return nil
}

func expectHTTPCode2(resp *http.Response, code int) error {
	if resp.StatusCode != code {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Trace(err)
		}

		return errors.Errorf(
			"HTTP Status Code: expected %d, got %d (body: %q)",
			code, resp.StatusCode, body,
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

type perUserData struct {
	id       int
	token    string
	username string
	email    string
}

// NOTE: perUserTestFunc shoult NOT take *testing.T argument, because this
// function should be able to run in parallel with others, and testing.T is not
// designed for that.
type perUserTestFunc func(si storage.Storage, be testBackend, u1, u2 *perUserData) error

func runPerUserTest(
	si storage.Storage, be testBackend, username1, email1, username2, email2 string, testFunc perUserTestFunc,
) error {
	var u1ID, u2ID int
	var u1Token, u2Token string
	var err error

	if u1ID, u1Token, err = testutils.CreateTestUser(si, username1, email1); err != nil {
		return errors.Trace(err)
	}
	be.UserCreated(u1ID, username1, u1Token)

	if u2ID, u2Token, err = testutils.CreateTestUser(si, username2, email2); err != nil {
		return errors.Trace(err)
	}
	be.UserCreated(u2ID, username2, u2Token)

	if err := testFunc(
		si, be,
		&perUserData{id: u1ID, token: u1Token, username: username1, email: email1},
		&perUserData{id: u2ID, token: u2Token, username: username2, email: email2},
	); err != nil {
		return errors.Trace(err)
	}

	if err := si.CheckIntegrity(); err != nil {
		return errors.Trace(err)
	}

	if err := be.DeleteUser(u1ID); err != nil {
		return errors.Trace(err)
	}

	if err := be.DeleteUser(u2ID); err != nil {
		return errors.Trace(err)
	}

	return nil
}
