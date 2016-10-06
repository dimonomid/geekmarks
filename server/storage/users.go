package storage

//go:generate go-bindata -nocompress -modtime 1 -mode 420 -pkg storage migrations

import (
	"database/sql"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"

	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

var (
	ErrUserDoesNotExist = errors.New("user does not exist")
)

// Either ID or Username should be given.
type GetUserArgs struct {
	ID       *int
	Username *string
}

type UserData struct {
	ID       int
	Username string
	Password string
	Email    string
}

func GetUser(tx *sql.Tx, args *GetUserArgs) (*UserData, error) {
	var ud UserData
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
			return nil, interror.WrapInternalError(err, ErrUserDoesNotExist)
		}
		// Some unexpected error
		return nil, hh.MakeInternalServerError(err)
	}

	return &ud, nil
}

func CreateUser(tx *sql.Tx, ud *UserData) (userID int, err error) {
	err = tx.QueryRow(
		"INSERT INTO users (username, password, email) VALUES ($1, $2, $3) RETURNING id",
		ud.Username, ud.Password, ud.Email,
	).Scan(&userID)
	if err != nil {
		return 0, hh.MakeInternalServerError(err)
	}

	// Also, create a root tag for the newly added user: NULL parent_id and an
	// empty string name
	_, err = CreateTag(tx, userID, 0 /*will be set to NULL*/, []string{""})
	if err != nil {
		return 0, errors.Trace(err)
	}

	return userID, nil
}
