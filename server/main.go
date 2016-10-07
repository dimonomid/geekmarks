package main

import (
	"database/sql"
	"flag"
	"net/http"

	"dmitryfrank.com/geekmarks/server/cptr"
	gmserver "dmitryfrank.com/geekmarks/server/server"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/golang/glog"
	"github.com/juju/errors"
)

func main() {
	flag.Parse()

	defer glog.Flush()

	err := gmserver.Initialize(true /*apply migrations*/)
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	err = createTestUsers()
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

// TODO: remove
func createTestUsers() error {
	err := storage.Tx(func(tx *sql.Tx) error {

		_, err := storage.GetUser(tx, &storage.GetUserArgs{
			ID: cptr.Int(1),
		})

		if err != nil {
			if errors.Cause(err) == storage.ErrUserDoesNotExist {
				glog.Infof("Creating test user: alice")
				_, err := storage.CreateUser(tx, &storage.UserData{
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

		_, err = storage.GetUser(tx, &storage.GetUserArgs{
			ID: cptr.Int(2),
		})

		if err != nil {
			if errors.Cause(err) == storage.ErrUserDoesNotExist {
				glog.Infof("Creating test user: bob")
				_, err := storage.CreateUser(tx, &storage.UserData{
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
