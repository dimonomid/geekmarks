// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

// +build !all_tests,!unit_tests,!integration_tests

package server

import (
	"errors"

	goji "goji.io"
	"goji.io/pat"
)

// Set endpoint which is used by tests only. This implementation is a no-op.
func setUserEndpointTest(
	pattern *pat.Pattern, gmh GMHandler, wsMux *WebSocketMux, mux *goji.Mux, gsu getSubjUser,
) {
	// Do nothing
}

func (gm *GMServer) testUserDelete(gmr *GMRequest) (resp interface{}, err error) {
	return nil, errors.New("testUserDelete is not built")
}
