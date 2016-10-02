package main

import (
	"flag"
	"net/http"

	goji "goji.io"
	"goji.io/pat"

	"dmitryfrank.com/geekmarks/server/middleware"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/golang/glog"
	"github.com/juju/errors"
)

func main() {
	flag.Parse()
	err := storage.Initialize()
	if err != nil {
		glog.Fatalf("%s\n", errors.Details(err))
	}

	rRoot := goji.NewMux()
	rRoot.Use(middleware.MakeLogger())

	rAPI := goji.SubMux()
	rRoot.Handle(pat.New("/api/*"), rAPI)

	rAPIMy := goji.SubMux()
	rAPI.Handle(pat.New("/my/*"), rAPIMy)
	rAPIMy.Use(authMiddleware)

	rAPIMy.HandleFunc(pat.Get("/test"), makeAPIHandler(testHandler))
	rAPI.HandleFunc(pat.Get("/test"), makeAPIHandler(testHandler))

	glog.Infof("Listening..")
	http.ListenAndServe(":4000", rRoot)
}

type testType struct {
	Username *string `json:"username"`
}

func testHandler(r *http.Request) (resp interface{}, err error) {
	var s string
	var sp *string
	v := r.Context().Value("username")
	if v != nil {
		s = v.(string)
		sp = &s
	}
	resp = testType{
		Username: sp,
	}

	return resp, nil
}
