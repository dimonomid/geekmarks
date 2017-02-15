// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package main

import (
	"flag"
	"fmt"
	"net/http"

	gmserver "dmitryfrank.com/geekmarks/server/server"
	storagecommon "dmitryfrank.com/geekmarks/server/storage/common"
	"github.com/golang/glog"
	"github.com/juju/errors"
)

var (
	port = flag.String("geekmarks.port", "8000", "Port to listen at.")
)

func main() {
	flag.Parse()

	defer glog.Flush()

	si, err := storagecommon.CreateStorage()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	err = si.Connect()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	err = si.ApplyMigrations()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	gminstance, err := gmserver.New(si)
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	handler, err := gminstance.CreateHandler()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	glog.Infof("Listening at the port %s ...", *port)
	http.ListenAndServe(fmt.Sprintf(":%s", *port), handler)
}
