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

func (s *StoragePostgres) GetUsers(tx *sql.Tx) ([]storage.UserData, error) {
	var ret []storage.UserData

	rows, err := tx.Query(
		"SELECT id, username, password, email FROM users",
	)
	if err != nil {
		return nil, hh.MakeInternalServerError(err)
	}

	defer rows.Close()
	for rows.Next() {
		var cur storage.UserData
		err := rows.Scan(&cur.ID, &cur.Username, &cur.Password, &cur.Email)
		if err != nil {
			return nil, hh.MakeInternalServerError(err)
		}
		ret = append(ret, cur)
	}

	return ret, nil
}

func (s *StoragePostgres) GetAccessToken(
	tx *sql.Tx, userID int, descr string, createIfNotExist bool,
) (token string, err error) {

	err = tx.QueryRow(
		"SELECT token FROM access_tokens WHERE user_id = $1 and descr = $2",
		userID, descr,
	).Scan(&token)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		// Some unexpected error
		return "", hh.MakeInternalServerError(err)
	}

	if token == "" {
		// Token does not exist
		if createIfNotExist {
			// Let's create one
			token = uniuri.NewLen(accessTokenLen)
			_, err := tx.Exec(
				"INSERT INTO access_tokens (user_id, token, descr) VALUES ($1, $2, $3)",
				userID, token, descr,
			)
			if err != nil {
				return "", interror.WrapInternalErrorf(
					err, "failed to create access token %q (%q, user_id: %d)",
					token, descr, userID,
				)
			}
		} else {
			return "", errors.Errorf("token with the descr %q does not exist", descr)
		}
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

func (s *StoragePostgres) GetUserByGoogleUserID(
	tx *sql.Tx, googleUserID string,
) (*storage.UserData, error) {
	var ud storage.UserData

	err := tx.QueryRow(`
SELECT u.id, u.username, u.password, u.email FROM users u
JOIN google_auth google ON google.user_id = u.id
WHERE google.google_user_id = $1`, googleUserID,
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

func (s *StoragePostgres) CreateGoogleUser(
	tx *sql.Tx, userID int, googleUserID, email string,
) error {
	_, err := tx.Exec(`
  INSERT INTO google_auth (google_user_id, user_id, email) VALUES ($1, $2, $3)
`, googleUserID, userID, email)
	if err != nil {
		return hh.MakeInternalServerError(err)
	}

	return nil
}
