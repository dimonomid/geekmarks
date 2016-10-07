package main

import (
	"database/sql"
	"flag"
	"net/http"

	"dmitryfrank.com/geekmarks/server/cptr"
	gmserver "dmitryfrank.com/geekmarks/server/server"
	"dmitryfrank.com/geekmarks/server/storage"
	storagecommon "dmitryfrank.com/geekmarks/server/storage/common"
	"github.com/golang/glog"
	"github.com/juju/errors"
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

	err = createTestUsers(si)
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	handler, err := gminstance.CreateHandler()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	glog.Infof("Listening..")
	http.ListenAndServe(":4000", handler)
}

// TODO: remove
func createTestUsers(si storage.Storage) error {
	err := si.Tx(func(tx *sql.Tx) error {

		_, err := si.GetUser(tx, &storage.GetUserArgs{
			ID: cptr.Int(1),
		})

		if err != nil {
			if errors.Cause(err) == storage.ErrUserDoesNotExist {
				glog.Infof("Creating test user: alice")
				_, err := si.CreateUser(tx, &storage.UserData{
					Username: "alice",
					Password: "alice",
					Email:    "alice@domain.com",
				})
				if err != nil {
					return errors.Trace(err)
				}
			} else {
				return errors.Trace(err)
			}
		}

		_, err = si.GetUser(tx, &storage.GetUserArgs{
			ID: cptr.Int(2),
		})

		if err != nil {
			if errors.Cause(err) == storage.ErrUserDoesNotExist {
				glog.Infof("Creating test user: bob")
				_, err := si.CreateUser(tx, &storage.UserData{
					Username: "bob",
					Password: "bob",
					Email:    "bob@domain.com",
				})
				if err != nil {
					return errors.Trace(err)
				}
			} else {
				return errors.Trace(err)
			}
		}

		return nil
	})

	return errors.Trace(err)
}
