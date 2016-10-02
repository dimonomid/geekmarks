package main

import (
	"encoding/json"
	"net/http"

	"github.com/golang/glog"
	"github.com/juju/errors"
)

var (
	internalServerError error
	unauthorizedError   error
)

func init() {
	internalServerError = errors.New("internal server error")
	unauthorizedError = errors.New("unauthorized")
}

type errorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

func respondWithError(w http.ResponseWriter, errResp error) {
	resp := errorResponse{
		Status:  getHTTPErrorCode(errResp),
		Message: errResp.Error(),
	}
	d, err := json.Marshal(resp)
	if err != nil {
		panic(err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.Status)
	_, err = w.Write(d)
	if err != nil {
		panic(err)
	}
}

func makeAPIHandler(
	f func(r *http.Request) (resp interface{}, err error),
) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := f(r)
		if err != nil {
			respondWithError(w, err)
			return
		}

		d, err := json.Marshal(resp)
		if err != nil {
			respondWithError(w, mkInternalServerError(err, "marshalling resp"))
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write(d)
		if err != nil {
			panic(err)
		}
	}
}

func mkInternalServerError(err error, message string) error {
	glog.Errorf("%s: %s", message, errors.Trace(err))
	if errors.Cause(err) != internalServerError {
		err = errors.Annotatef(internalServerError, message)
	}
	return err
}

func getHTTPErrorCode(err error) int {
	status := http.StatusBadRequest

	switch errors.Cause(err) {
	case internalServerError:
		status = http.StatusInternalServerError
	case unauthorizedError:
		status = http.StatusUnauthorized
	}

	return status
}
