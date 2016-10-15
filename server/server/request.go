package server

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"goji.io/pattern"

	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
)

type GMRequest struct {
	SubjUser *storage.UserData
	Caller   *storage.UserData
	Method   string
	Path     string
	Values   map[string][]string
	Body     io.ReadCloser
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
		SubjUser: subjUser,
		Caller:   getAuthnUserDataByReq(r),
		Method:   r.Method,
		Path:     pattern.Path(r.Context()),
		Values:   map[string][]string(r.Form),
		Body:     ioutil.NopCloser(bytes.NewReader(b.Bytes())),
	}

	return gmr, nil
}

func makeGMRequestFromWebSocketRequest(
	wsr *WebSocketRequest, caller *storage.UserData, subjUser *storage.UserData,
	path string,
) (*GMRequest, error) {
	values := map[string][]string{}
	for k, v := range wsr.Values {
		switch val := v.(type) {
		case string:
			values[k] = []string{val}
		case []string:
			values[k] = val
		default:
			return nil, errors.Errorf("value can only be a string or an array of strings")
		}
	}

	bodyData, err := json.Marshal(wsr.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	gmr := &GMRequest{
		SubjUser: subjUser,
		Caller:   caller,
		Method:   wsr.Method,
		Path:     path,
		Values:   values,
		Body:     ioutil.NopCloser(bytes.NewReader(bodyData)),
	}

	return gmr, nil
}
