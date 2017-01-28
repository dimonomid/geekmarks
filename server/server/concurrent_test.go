// +build all_tests integration_tests

package server

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
)

var (
	concurrentTests = []perUserTestFunc{
		perUserTestTagsMovingDelLeafs,
		perUserTestTagsMovingKeepLeafs,
	}
)

func TestConcurrent(t *testing.T) {
	goroutinesCnt := 16
	testsCnt := 32

	runTests := func(
		si storage.Storage, be testBackend, username, email string,
		errChan chan<- error,
	) {
		for i := 0; i < testsCnt; i++ {
			var testFunc perUserTestFunc

			// Pick a random test to run
			testFunc = concurrentTests[rand.Intn(len(concurrentTests))]

			// Run it
			err := runPerUserTest(si, be, username, email, testFunc)
			if err != nil {
				errChan <- errors.Trace(err)
				return
			}
		}

		errChan <- nil
	}

	rand.Seed(time.Now().UTC().UnixNano())

	// TODO: implement random backend, so that it will randomly use ws or http,
	// and "users" or "my" endpoint
	be := makeTestBackendHTTP(t, testBackendOpts{
		UseWS: true,
	})

	runWithRealDBAndBackend(t, be, func(si storage.Storage, be testBackend) error {
		errChan := make(chan error)

		// Create needed amount of goroutines which execute random tests
		for i := 0; i < goroutinesCnt; i++ {
			go runTests(
				si, be, fmt.Sprintf("test%d", i), fmt.Sprintf("%d@test.test", i),
				errChan,
			)
		}

		// Wait for all of them to finish, and gather results
		errs := []error{}
		for i := 0; i < goroutinesCnt; i++ {
			errs = append(errs, <-errChan)
		}

		// Check results
		for _, err := range errs {
			if err != nil {
				return errors.Trace(err)
			}
		}

		return nil
	})
}
