package main

import (
	"net/http"

	"dmitryfrank.com/geekmarks/server/storage"
)

type authzArgs struct {
	OwnerID int
}

func authorizeOperation(r *http.Request, args *authzArgs) (ok bool, err error) {
	// We might have much more fields in authzArgs in the future, but for now
	// we keep things simple: only owner can do everything with their data;
	// others can do nothing.

	v := r.Context().Value("authUserData")
	if v == nil {
		// Not authenticated
		return false, nil
	}

	ud := v.(*storage.UserData)

	if ud.ID != args.OwnerID {
		// Another user
		return false, nil
	}

	return true, nil
}
