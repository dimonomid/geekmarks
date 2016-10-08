package storage

import (
	"database/sql"

	"github.com/juju/errors"
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

type TagData struct {
	ID          int
	OwnerID     int
	ParentTagID int
	Description string
	Names       []string
}

type Storage interface {
	//-- Common
	Connect() error
	ApplyMigrations() error
	Tx(fn func(*sql.Tx) error) error
	//-- Users
	GetUser(tx *sql.Tx, args *GetUserArgs) (*UserData, error)
	CreateUser(tx *sql.Tx, ud *UserData) (userID int, err error)
	//-- Tags
	CreateTag(tx *sql.Tx, td *TagData) (tagID int, err error)
	GetTagIDByPath(tx *sql.Tx, ownerID int, tagPath string) (int, error)
	GetTagOwnerByID(tx *sql.Tx, tagID int) (ownerID int, err error)
	GetTagIDByName(tx *sql.Tx, parentTagID int, tagName string) (int, error)
	GetRootTagID(tx *sql.Tx, ownerID int) (int, error)
}
