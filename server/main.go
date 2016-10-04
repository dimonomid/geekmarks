package main

import (
	"flag"
	"net/http"

	goji "goji.io"
	"goji.io/pat"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/middleware"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/golang/glog"
	"github.com/juju/errors"
)

func main() {
	flag.Parse()

	defer glog.Flush()

	err := storage.Initialize()
	if err != nil {
		glog.Fatalf("%s\n", errors.Details(err))
	}

	rRoot := goji.NewMux()
	rRoot.Use(middleware.MakeLogger())

	rAPI := goji.SubMux()
	rRoot.Handle(pat.New("/api/*"), rAPI)
	{
		rAPI.Use(hh.MakeDesiredContentTypeMiddleware("application/json"))
		// We use authnMiddleware here and not on the root router above, since we
		// need hh.MakeDesiredContentTypeMiddleware to go before it.
		rAPI.Use(authnMiddleware)

		rAPIMy := goji.SubMux()
		rAPI.Handle(pat.New("/my/*"), rAPIMy)
		{
			// "my" endpoints don't make sense for non-authenticated users
			rAPIMy.Use(authnRequiredMiddleware)

			rAPIMy.HandleFunc(pat.Get("/test"), hh.MakeAPIHandler(testHandler))
		}
		rAPI.HandleFunc(pat.Get("/test"), hh.MakeAPIHandler(testHandler))
	}

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
