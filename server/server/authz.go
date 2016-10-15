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

func (gm *GMServer) authorizeOperationByReq(
	r *http.Request, args *authzArgs,
) error {
	callerData := getAuthnUserDataByReq(r)

	return gm.authorizeOperation(callerData, args)
}

func (gm *GMServer) authorizeOperation(
	callerData *storage.UserData, args *authzArgs,
) error {
	// We might have much more fields in authzArgs in the future, but for now
	// we keep things simple: only owner can do everything with their data;
	// others can do nothing.

	if callerData.ID != args.OwnerID {
		// Another user
		return hh.MakeForbiddenError()
	}

	return nil
}

// The OwnerID field in args is overwritten by the user data returned by
// gu, so at the moment clients have to call this function with just
// &authzArgs{}, but we'll probably have more fields in the future, so,
// this function still takes this argument.
func (gm *GMServer) getUserAndAuthorizeByReq(
	r *http.Request, gu getUser, args *authzArgs,
) (*storage.UserData, error) {
	ud, err := gu(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	aArgs := *args
	aArgs.OwnerID = ud.ID

	err = gm.authorizeOperationByReq(r, &aArgs)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return ud, nil
}
