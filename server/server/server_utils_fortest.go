// This file contains test-specific implementation of some functions.  I'd
// rather name it "server_utils_test.go", but unfortunately it doesn't work: go
// tool complains about undefined symbols (which are actually defined here).

// +build all_tests unit_tests integration_tests

package server

import (
	"database/sql"

	"github.com/juju/errors"
	goji "goji.io"
	"goji.io/pat"
)

// Set endpoint which is used by tests only.
func setUserEndpointTest(
	pattern *pat.Pattern, gmh GMHandler, wsMux *WebSocketMux, mux *goji.Mux, gsu getSubjUser,
) {
	setUserEndpoint(pattern, gmh, wsMux, mux, gsu)
}

type testUserDeleteResp struct {
}

func (gm *GMServer) testUserDelete(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	err = gm.si.Tx(func(tx *sql.Tx) error {
		err = gm.si.DeleteUser(tx, gmr.SubjUser.ID)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	// Invalidate tree cache for the user
	userIDToTagsTree.DeleteCacheForUser(gmr.SubjUser.ID)

	resp = testUserDeleteResp{}

	return resp, nil
}
