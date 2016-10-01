package main

import (
	"flag"
	"time"

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

	for {
		time.Sleep(1 * time.Second)
		glog.Infof("hey")
	}
}
