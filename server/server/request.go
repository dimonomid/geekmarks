package server

import (
	"io"
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
		Body:     r.Body,
	}

	return gmr, nil
}
