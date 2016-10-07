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

func (gm *GMServer) authorizeOperation(r *http.Request, args *authzArgs) (ok bool, err error) {
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
// gu, so at the moment clients have to call this function with just
// &authzArgs{}, but we'll probably have more fields in the future, so,
// this function still takes this argument.
func (gm *GMServer) getUserAndAuthorize(
	r *http.Request, gu getUser, args *authzArgs,
) (*storage.UserData, error) {
	ud, err := gu(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	aArgs := *args
	aArgs.OwnerID = ud.ID

	authorized, err := gm.authorizeOperation(r, &aArgs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if !authorized {
		return nil, hh.MakeForbiddenError()
	}

	return ud, nil
}
