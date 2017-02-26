// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package httphelper // import "dmitryfrank.com/geekmarks/server/httphelper"

import (
	"context"
	"encoding/json"
	"net/http"

	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/middleware"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

var (
	internalServerError error
	unauthorizedError   error
	forbiddenError      error
	notImplementedError error
)

func init() {
	internalServerError = errors.New("internal server error")
	unauthorizedError = errors.New("unauthorized")
	forbiddenError = errors.New("forbidden")
	notImplementedError = errors.New("not implemented")
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

const (
	DesiredContentTypeKey = "desiredContentType"
)

func GetErrorStruct(errResp error) *ErrorResponse {
	httpErrorCode := GetHTTPErrorCode(errResp)
	return &ErrorResponse{
		Status:  httpErrorCode,
		Message: errResp.Error(),
	}
}

func RespondWithError(w http.ResponseWriter, r *http.Request, errResp error) {
	errStruct := GetErrorStruct(errResp)

	desiredContentType := "text/html"

	if errors.Cause(errResp) == internalServerError {
		glog.Errorf("INTERNAL SERVER ERROR:\n" + interror.ErrorStack(errResp))
	} else {
		glog.V(2).Infof(errors.ErrorStack(errResp))
	}

	v := r.Context().Value(DesiredContentTypeKey)
	if v != nil {
		var ok bool
		desiredContentType, ok = v.(string)
		if !ok {
			glog.Errorf("wrong type of desiredContentType: %T (%v)",
				desiredContentType, desiredContentType)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}

	switch desiredContentType {
	case "application/json":
		d, err := json.MarshalIndent(errStruct, "", "  ")
		if err != nil {
			panic(err)
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(errStruct.Status)
		_, err = w.Write(d)
		if err != nil {
			panic(err)
		}
	case "text/html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(errStruct.Status)
		_, err := w.Write([]byte("Error: " + errResp.Error()))
		if err != nil {
			panic(err)
		}
	default:
		glog.Errorf("wrong desiredContentType: %q", desiredContentType)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func MakeAPIHandler(
	f func(r *http.Request) (resp interface{}, err error),
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := f(r)
		if err != nil {
			RespondWithError(w, r, errors.Trace(err))
			return
		}

		d, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			RespondWithError(w, r, MakeInternalServerError(
				errors.Annotatef(err, "marshalling resp"),
			))
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(d)
		if err != nil {
			panic(err)
		}
	}
}

func MakeAPIHandlerWWriter(
	f func(w http.ResponseWriter, r *http.Request) (err error),
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			//TODO: it writes the headers again!
			RespondWithError(w, r, errors.Trace(err))
			return
		}
	}
}

// MakeInternalServerError logs the given error and returns internalServerError
// annotated with the message, which does NOT wrap the original error, since we
// don't want internal server error details to percolate to clients.
func MakeInternalServerError(intError error) error {
	if errors.Cause(intError) != internalServerError {
		return interror.WrapInternalError(intError, internalServerError)
	}
	return errors.Trace(intError)
}

func MakeInternalServerErrorf(
	intError error, format string, args ...interface{},
) error {
	return interror.WrapInternalError(
		intError,
		errors.Annotatef(internalServerError, format, args...),
	)
}

func MakeUnauthorizedError() error {
	return unauthorizedError
}

func MakeForbiddenError() error {
	return forbiddenError
}

func MakeNotImplementedError() error {
	return notImplementedError
}

func GetHTTPErrorCode(err error) int {
	status := http.StatusBadRequest

	switch errors.Cause(err) {
	case internalServerError:
		status = http.StatusInternalServerError
	case unauthorizedError:
		status = http.StatusUnauthorized
	case forbiddenError:
		status = http.StatusForbidden
	case notImplementedError:
		status = http.StatusNotAcceptable
	}

	return status
}

func MakeDesiredContentTypeMiddleware(
	contentType string,
) func(inner http.Handler) http.Handler {
	return func(inner http.Handler) http.Handler {
		mw := func(w http.ResponseWriter, r *http.Request) {
			// Process request
			inner.ServeHTTP(w, r.WithContext(context.WithValue(
				r.Context(), DesiredContentTypeKey, contentType,
			)))
		}
		return middleware.MkMiddleware(mw)
	}
}
