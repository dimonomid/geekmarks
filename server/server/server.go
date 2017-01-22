package server

import (
	"database/sql"
	"flag"
	"fmt"
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

var googleOAuthCredsFile = flag.String(
	"google_oauth_creds_file", "",
	"Path to the file with Google app ID and secret.",
)

const (
	BookmarkID = "bkmid"

	providerGoogle = "google"
)

type GMServer struct {
	si             storage.Storage
	wsMux          *WebSocketMux
	oauthProviders map[string]*OAuthCreds
}

func New(si storage.Storage) (*GMServer, error) {
	oauthProviders := map[string]*OAuthCreds{}

	if *googleOAuthCredsFile != "" {
		googleOAuthCreds, err := ReadOAuthCredsFile(*googleOAuthCredsFile)
		if err != nil {
			return nil, errors.Trace(err)
		}

		oauthProviders[providerGoogle] = googleOAuthCreds
	} else {
		// Google creds file was not provided: Google auth is disabled
		oauthProviders[providerGoogle] = nil
	}

	gm := GMServer{
		si:             si,
		wsMux:          &WebSocketMux{},
		oauthProviders: oauthProviders,
	}
	return &gm, nil
}

func setUserEndpoint(
	pattern *pat.Pattern, gmh GMHandler, wsMux *WebSocketMux, mux *goji.Mux, gsu getSubjUser,
) {
	mkUserHandler := func(
		uh func(gmr *GMRequest) (resp interface{}, err error),
		gsu getSubjUser,
	) func(r *http.Request) (resp interface{}, err error) {
		return func(r *http.Request) (resp interface{}, err error) {
			gmr, err := makeGMRequestFromHttpRequest(r, gsu)
			if err != nil {
				return nil, errors.Trace(err)
			}
			return uh(gmr)
		}
	}

	handler := hh.MakeAPIHandler(mkUserHandler(gmh, gsu))
	mux.HandleFunc(pattern, handler)

	if wsMux != nil {
		wsMux.Add(pattern, gmh)
	}
}

type GMHandler func(gmr *GMRequest) (resp interface{}, err error)

func (gm *GMServer) CreateHandler() (http.Handler, error) {
	rRoot := goji.NewMux()
	rRoot.Use(middleware.MakeLogger())

	rAPI := goji.SubMux()
	rRoot.Handle(pat.New("/api/*"), rAPI)
	{
		rAPI.Use(hh.MakeDesiredContentTypeMiddleware("application/json"))
		// We use authnMiddleware here and not on the root router above, since we
		// need hh.MakeDesiredContentTypeMiddleware to go before it.
		rAPI.Use(gm.authnMiddleware)

		rAPIUsers := goji.SubMux()
		rAPI.Handle(pat.New("/users/:userid/*"), rAPIUsers)
		{
			gm.setupUserAPIEndpoints(rAPIUsers, gm.getUserFromURLParam)
		}

		rAPIMy := goji.SubMux()
		rAPI.Handle(pat.New("/my/*"), rAPIMy)
		{
			// "my" endpoints don't make sense for non-authenticated users
			rAPIMy.Use(gm.authnRequiredMiddleware)

			gm.setupUserAPIEndpoints(rAPIMy, gm.getUserFromAuthn)
		}

		rAPIAuth := goji.SubMux()
		rAPI.Handle(pat.New("/auth/:provider/*"), rAPIAuth)
		{
			gm.setupAuthAPIEndpoints(rAPIAuth, gm.getUserFromAuthnIfExists)
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

	return rRoot, nil
}

type getSubjUser func(r *http.Request) (*storage.UserData, error)

// Sets up user-related endpoints at a given mux. We need this function since
// we have two ways to access user data: through the "/api/users/:userid" and
// through the shortcut "/api/my"; so, in order to avoid duplication, this
// function sets up everything given the function that gets user data.
func (gm *GMServer) setupUserAPIEndpoints(mux *goji.Mux, gsu getSubjUser) {
	mkUserHandlerWWriter := func(
		uh func(w http.ResponseWriter, r *http.Request, gsu getSubjUser, gmh GMHandler) (err error),
		gsu getSubjUser, gmh GMHandler,
	) func(w http.ResponseWriter, r *http.Request) (err error) {
		return func(w http.ResponseWriter, r *http.Request) (err error) {
			return uh(w, r, gsu, gmh)
		}
	}

	setUserEndpoint(pat.Get("/tags"), gm.userTagsGet, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Get("/tags/*"), gm.userTagsGet, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Post("/tags"), gm.userTagsPost, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Post("/tags/*"), gm.userTagsPost, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Delete("/tags"), gm.userTagDelete, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Delete("/tags/*"), gm.userTagDelete, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Put("/tags"), gm.userTagPut, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Put("/tags/*"), gm.userTagPut, gm.wsMux, mux, gsu)

	setUserEndpoint(pat.Get("/bookmarks"), gm.userBookmarksGet, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Post("/bookmarks"), gm.userBookmarksPost, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Get("/bookmarks/:"+BookmarkID), gm.userBookmarkGet, gm.wsMux, mux, gsu)
	setUserEndpoint(pat.Put("/bookmarks/:"+BookmarkID), gm.userBookmarkPut, gm.wsMux, mux, gsu)

	setUserEndpoint(pat.Get("/add_test_tags_tree"), gm.addTestTagsTree, gm.wsMux, mux, gsu)

	{
		handler := hh.MakeAPIHandlerWWriter(
			mkUserHandlerWWriter(gm.webSocketConnect, gsu, gm.wsMux.Handle),
		)
		mux.HandleFunc(pat.Get("/wsconnect"), handler)
	}

}

// Retrieves user data from the userid given in an URL, like "123" in
// "/api/users/123/foo/bar"
func (gm *GMServer) getUserFromURLParam(r *http.Request) (*storage.UserData, error) {
	useridStr := pat.Param(r, "userid")
	userid, err := strconv.Atoi(useridStr)
	if err != nil {
		return nil, errors.Errorf("invalid user id: %q", useridStr)
	}

	var ud *storage.UserData
	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		ud, err = gm.si.GetUser(tx, &storage.GetUserArgs{
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
func (gm *GMServer) getUserFromAuthn(r *http.Request) (*storage.UserData, error) {
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

// Like getUserFromAuthn, but the user is allowed to be nil
func (gm *GMServer) getUserFromAuthnIfExists(r *http.Request) (*storage.UserData, error) {
	// authUserData should always be present here thanks to
	// authnRequiredMiddleware
	ud := r.Context().Value("authUserData")
	if ud == nil {
		return nil, nil
	}

	return ud.(*storage.UserData), nil
}

func getErrorMsgParamRequired(param string, values []string) string {
	return fmt.Sprintf("parameter required: %q, possible values: %q", param, values)
}
