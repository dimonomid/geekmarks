package main

import (
	"flag"
	"net/http"

	gmserver "dmitryfrank.com/geekmarks/server/server"
	"github.com/golang/glog"
	"github.com/juju/errors"
)

func main() {
	flag.Parse()

	defer glog.Flush()

	err := gmserver.Initialize()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	handler, err := gmserver.CreateHandler()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	glog.Infof("Listening..")
	http.ListenAndServe(":4000", handler)
}
