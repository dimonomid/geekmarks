package main

import (
	"database/sql"
	"flag"
	"net/http"
	"strconv"

	goji "goji.io"
	"goji.io/pat"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/middleware"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/golang/glog"
	"github.com/juju/errors"
)

func main() {
	flag.Parse()

	defer glog.Flush()

	err := storage.Initialize()
	if err != nil {
		glog.Fatalf("%s\n", errors.ErrorStack(err))
	}

	rRoot := goji.NewMux()
	rRoot.Use(middleware.MakeLogger())

	rAPI := goji.SubMux()
	rRoot.Handle(pat.New("/api/*"), rAPI)
	{
		rAPI.Use(hh.MakeDesiredContentTypeMiddleware("application/json"))
		// We use authnMiddleware here and not on the root router above, since we
		// need hh.MakeDesiredContentTypeMiddleware to go before it.
		rAPI.Use(authnMiddleware)

		rAPIUsers := goji.SubMux()
		rAPI.Handle(pat.New("/users/:userid/*"), rAPIUsers)
		{
			setupUserAPIEndpoints(rAPIUsers, getUserFromURLParam)
		}

		rAPIMy := goji.SubMux()
		rAPI.Handle(pat.New("/my/*"), rAPIMy)
		{
			// "my" endpoints don't make sense for non-authenticated users
			rAPIMy.Use(authnRequiredMiddleware)

			setupUserAPIEndpoints(rAPIMy, getUserFromAuthn)
		}

		rAPI.HandleFunc(
			pat.Get("/test_internal_error"), hh.MakeAPIHandler(
				func(r *http.Request) (resp interface{}, err error) {
					errTest := errors.Errorf("some private error")
					errTest = errors.Annotatef(errTest, "private annotation")
					return nil, errors.Annotatef(hh.MakeInternalServerError(errTest), "public annotation")
				},
			),
		)
	}

	glog.Infof("Listening..")
	http.ListenAndServe(":4000", rRoot)
}

type GetUser func(r *http.Request) (*storage.UserData, error)

// Sets up user-related endpoints at a given mux. We need this function since
// we have two ways to access user data: through the "/api/users/:userid" and
// through the shortcut "/api/my"; so, in order to avoid duplication, this
// function sets up everything given the function that gets user data.
func setupUserAPIEndpoints(mux *goji.Mux, getUser GetUser) {
	mkUserHandler := func(
		uh func(r *http.Request, getUser GetUser) (resp interface{}, err error),
		getUser GetUser,
	) func(r *http.Request) (resp interface{}, err error) {
		return func(r *http.Request) (resp interface{}, err error) {
			return uh(r, getUser)
		}
	}

	mux.HandleFunc(
		pat.Get("/tags"), hh.MakeAPIHandler(mkUserHandler(userTagsGet, getUser)),
	)

	mux.HandleFunc(
		pat.Post("/tags"), hh.MakeAPIHandler(mkUserHandler(userTagsPost, getUser)),
	)
}

// Retrieves user data from the userid given in an URL, like "123" in
// "/api/users/123/foo/bar"
func getUserFromURLParam(r *http.Request) (*storage.UserData, error) {
	useridStr := pat.Param(r, "userid")
	userid, err := strconv.Atoi(useridStr)
	if err != nil {
		return nil, errors.Errorf("invalid user id: %q", useridStr)
	}

	var ud *storage.UserData
	err = storage.Tx(func(tx *sql.Tx) error {
		var err error
		ud, err = storage.GetUser(tx, &storage.GetUserArgs{
			ID: cptr.Int(userid),
		})
		return errors.Trace(err)
	})
	if err != nil {
		glog.Errorf(
			"Failed to get user with id %d (from URL param): %s", userid, err,
		)
		return nil, errors.Errorf("invalid user id: %q", useridStr)
	}

	return ud, nil
}

// Retrieves user data from the authentication data
func getUserFromAuthn(r *http.Request) (*storage.UserData, error) {
	// authUserData should always be present here thanks to
	// authnRequiredMiddleware
	ud := r.Context().Value("authUserData")
	if ud == nil {
		return nil, hh.MakeInternalServerError(
			errors.Errorf("authUserData is nil but it should not be"),
		)
	}

	return ud.(*storage.UserData), nil
}
