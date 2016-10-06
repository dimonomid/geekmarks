package server

import (
	"net/http"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
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

// The OwnerID field in args is overwritten by the user data returned by
// getUser, so at the moment clients have to call this function with just
// &authzArgs{}, but we'll probably have more fields in the future, so,
// this function still takes this argument.
func getUserAndAuthorize(
	r *http.Request, getUser GetUser, args *authzArgs,
) (*storage.UserData, error) {
	ud, err := getUser(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	aArgs := *args
	aArgs.OwnerID = ud.ID

	authorized, err := authorizeOperation(r, &aArgs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if !authorized {
		return nil, hh.MakeForbiddenError()
	}

	return ud, nil
}
