// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"github.com/dimonomid/interrors"

	"goji.io/pat"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"github.com/juju/errors"
)

type WebSocketRequest struct {
	Id     int    `json:"id"`
	Method string `json:"method"`
	// Path after user address: e.g. the replica of "/api/my/tags" is "/tags".
	Path   string                 `json:"path"`
	Values map[string]interface{} `json:"values"`
	Body   interface{}            `json:"body,omitempty"`
}

type WebSocketResponse struct {
	Id     int    `json:"id"`
	Method string `json:"method"`
	// Path after user address: e.g. the replica of "/api/my/tags" is "/tags".
	Path   string                 `json:"path"`
	Values map[string]interface{} `json:"values,omitempty"`
	Status int                    `json:"status"`
	Body   interface{}            `json:"body"`
}

type route struct {
	pattern *pat.Pattern
	handler GMHandler
}

type WebSocketMux struct {
	routes []route
}

func (m *WebSocketMux) Add(pattern *pat.Pattern, handler GMHandler) {
	m.routes = append(m.routes, route{
		pattern: pattern,
		handler: handler,
	})
}

func (m *WebSocketMux) Handle(gmr *GMRequest) (resp interface{}, err error) {
	for _, route := range m.routes {
		if r2 := route.pattern.Match(gmr.HttpReq); r2 != nil {
			gmr.HttpReq = r2
			resp, err = route.handler(gmr)
			if err != nil {
				return nil, errors.Trace(err)
			}
			break
		}
	}

	if resp == nil {
		// TODO: use specific error for which hh.GetHTTPErrorCode would return 404
		return nil, errors.Errorf("page not found")
	}

	return resp, nil
}

func parseWebSocketRequest(reader io.Reader) (*WebSocketRequest, error) {
	var wsr *WebSocketRequest
	decoder := json.NewDecoder(reader)
	// UseNumber is needed to prevent from interpeting numbers in body as floats,
	// and re-marshalling them in scientific notation (in case the number is
	// large)
	decoder.UseNumber()
	err := decoder.Decode(&wsr)
	if err != nil {
		return nil, interrors.WrapInternalError(
			err,
			errors.Errorf("invalid data"),
		)
	}

	return wsr, nil
}

func mkHttpReqFromWebSocketReq(wsr *WebSocketRequest) (*http.Request, error) {
	r := http.Request{}
	switch wsr.Method {
	case "GET":
		r.Method = wsr.Method
	case "POST":
		r.Method = wsr.Method
	case "PUT":
		r.Method = wsr.Method
	case "DELETE":
		r.Method = wsr.Method
	default:
		return nil, errors.Errorf("wrong method %q", wsr.Method)
	}

	r.URL = &url.URL{
		Path: wsr.Path,
	}

	bodyData, err := json.Marshal(wsr.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	r.Body = ioutil.NopCloser(bytes.NewReader(bodyData))

	// TODO: Probably set Form values as well

	return &r, nil
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (gm *GMServer) webSocketConnect(
	w http.ResponseWriter,
	r *http.Request,
	gsu getSubjUser,
	wsMux GMHandler,
) error {
	subjUser, err := gm.getUserAndAuthorizeByReq(r, gsu, &authzArgs{})
	if err != nil {
		return errors.Trace(err)
	}

	caller := getAuthnUserDataByReq(r)

	fmt.Println("subj user:", subjUser)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return errors.Trace(err)
	}

	go func() (err error) {
		defer func() {
			glog.Infof(
				"Websocket goroutine for the user %s exits: %s",
				subjUser.Email, err,
			)
		}()
		for {
			messageType, reader, err := conn.NextReader()
			if err != nil {
				return errors.Trace(err)
			}

			// Start timer
			start := time.Now()

			status := http.StatusOK

			// Here we define and call this intermediary function, because the error
			// which happens there is not considered fatal: instead, it is reported
			// back to the client.
			resp, wsr, err := func() (resp interface{}, wsr *WebSocketRequest, err error) {
				wsr, err = parseWebSocketRequest(reader)
				if err != nil {
					return nil, wsr, errors.Trace(err)
				}

				gmr, err := makeGMRequestFromWebSocketRequest(
					wsr, caller, subjUser,
				)
				if err != nil {
					return nil, wsr, errors.Trace(err)
				}

				resp, err = wsMux(gmr)
				if err != nil {
					return nil, wsr, errors.Trace(err)
				}

				return resp, wsr, nil
			}()
			if err != nil {
				errResp := hh.GetErrorStruct(err)
				status = errResp.Status
				resp = errResp
			}

			if wsr == nil {
				wsr = &WebSocketRequest{}
			}

			fullResp := WebSocketResponse{
				Id:     wsr.Id,
				Method: wsr.Method,
				Path:   wsr.Path,
				Values: wsr.Values,
				Status: status,
				Body:   resp,
			}

			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			w, err := conn.NextWriter(messageType)
			if err != nil {
				return errors.Trace(err)
			}

			encoder := json.NewEncoder(w)
			err = encoder.Encode(fullResp)
			if err != nil {
				return errors.Trace(err)
			}
			if err := w.Close(); err != nil {
				return errors.Trace(err)
			}

			glog.Infof("%v: %13v", wsr, latency)
		}
	}()

	return nil
}
