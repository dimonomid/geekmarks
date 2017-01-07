package postgres

import (
	"database/sql"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"

	"github.com/dchest/uniuri"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

const (
	accessTokenLen = 32
)

func (s *StoragePostgres) GetUser(
	tx *sql.Tx, args *storage.GetUserArgs,
) (*storage.UserData, error) {
	var ud storage.UserData
	queryArgs := []interface{}{}
	where := ""
	if args.ID != nil {
		where = "id = $1"
		queryArgs = append(queryArgs, *args.ID)
	} else if args.Username != nil {
		where = "username = $1"
		queryArgs = append(queryArgs, *args.Username)
	} else {
		return nil, hh.MakeInternalServerError(errors.Errorf(
			"neither id nor username is given to storage.GetUser()",
		))
	}

	err := tx.QueryRow(
		"SELECT id, username, password, email FROM users WHERE "+where,
		queryArgs...,
	).Scan(&ud.ID, &ud.Username, &ud.Password, &ud.Email)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// TODO: annotate error with the id or name
			return nil, interror.WrapInternalError(err, storage.ErrUserDoesNotExist)
		}
		// Some unexpected error
		return nil, hh.MakeInternalServerError(err)
	}

	return &ud, nil
}

func (s *StoragePostgres) CreateUser(
	tx *sql.Tx, ud *storage.UserData,
) (userID int, err error) {
	err = tx.QueryRow(
		"INSERT INTO users (username, password, email) VALUES ($1, $2, $3) RETURNING id",
		ud.Username, ud.Password, ud.Email,
	).Scan(&userID)
	if err != nil {
		return 0, hh.MakeInternalServerError(err)
	}

	// Also, create a root tag for the newly added user: NULL parent_id and an
	// empty string name
	_, err = s.CreateTag(tx, &storage.TagData{
		OwnerID:     userID,
		Description: cptr.String("Root pseudo-tag"),
		Names:       []string{""},
	})
	if err != nil {
		return 0, errors.Trace(err)
	}

	return userID, nil
}

func (s *StoragePostgres) CreateAccessToken(
	tx *sql.Tx, userID int, token string,
) (string, error) {

	// if given token is an empty string, generate a random token
	if token == "" {
		token = uniuri.NewLen(accessTokenLen)
	}

	_, err := tx.Exec(
		"INSERT INTO access_tokens (user_id, token) VALUES ($1, $2)",
		userID, token,
	)
	if err != nil {
		return "", interror.WrapInternalErrorf(err, "failed to create access token %q", token)
	}

	return token, nil
}

func (s *StoragePostgres) GetUserByAccessToken(
	tx *sql.Tx, token string,
) (*storage.UserData, error) {
	var ud storage.UserData

	err := tx.QueryRow(`
SELECT u.id, u.username, u.password, u.email FROM users u
JOIN access_tokens tok ON tok.user_id = u.id
WHERE tok.token = $1`, token,
	).Scan(&ud.ID, &ud.Username, &ud.Password, &ud.Email)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			// TODO: annotate error with the id or name
			return nil, hh.MakeUnauthorizedError()
		}
		// Some unexpected error
		return nil, hh.MakeInternalServerError(err)
	}

	return &ud, nil
}
