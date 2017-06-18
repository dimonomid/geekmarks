// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"goji.io/pattern"

	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
)

type GMRequest struct {
	HttpReq  *http.Request
	SubjUser *storage.UserData
	Caller   *storage.UserData
	// TODO: remove Method from here, use HttpReq.Method instead
	//       (it is already populated correctly from websocket request)
	Method string
	Values map[string][]string
	Body   io.ReadCloser
}

func (gmr *GMRequest) FormValue(key string) string {
	if vs := gmr.Values[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

func makeGMRequestFromHttpRequest(
	r *http.Request, gsu getSubjUser,
) (*GMRequest, error) {
	// calling r.ParseForm clears r.Body, so we have to copy body data
	var b bytes.Buffer

	io.Copy(&b, r.Body)

	err := r.ParseForm()
	if err != nil {
		return nil, errors.Trace(err)
	}

	subjUser, err := gsu(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	gmr := &GMRequest{
		HttpReq:  r,
		SubjUser: subjUser,
		Caller:   getAuthnUserDataByReq(r),
		Method:   r.Method,
		Values:   map[string][]string(r.Form),
		Body:     ioutil.NopCloser(bytes.NewReader(b.Bytes())),
	}

	return gmr, nil
}

func makeGMRequestFromWebSocketRequest(
	wsr *WebSocketRequest, caller *storage.UserData, subjUser *storage.UserData,
) (*GMRequest, error) {
	values := map[string][]string{}
	for k, v := range wsr.Values {
		switch val := v.(type) {
		case string:
			values[k] = []string{val}
		case json.Number:
			values[k] = []string{val.String()}
		case []interface{}:
			values[k] = []string{}
			for _, v := range val {
				switch ival := v.(type) {
				case string:
					values[k] = append(values[k], ival)
				case json.Number:
					values[k] = append(values[k], ival.String())
				default:
					return nil, errors.Errorf("value can only be a string, a number, or an array of these")
				}
			}
		default:
			return nil, errors.Errorf("value can only be a string, a number, or an array of these", v)
		}
	}

	bodyData, err := json.Marshal(wsr.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	httpReq, err := mkHttpReqFromWebSocketReq(wsr)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ctx := context.Background()
	ctx = pattern.SetPath(ctx, httpReq.URL.EscapedPath())
	httpReq = httpReq.WithContext(ctx)

	gmr := &GMRequest{
		HttpReq:  httpReq,
		SubjUser: subjUser,
		Caller:   caller,
		Method:   wsr.Method,
		Values:   values,
		Body:     ioutil.NopCloser(bytes.NewReader(bodyData)),
	}

	return gmr, nil
}
