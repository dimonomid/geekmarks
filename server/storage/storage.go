package storage

import (
	"database/sql"
	"strconv"
	"strings"
	"unicode"

	"github.com/juju/errors"
)

var (
	ErrUserDoesNotExist = errors.New("user does not exist")
	ErrTagDoesNotExist  = errors.New("tag does not exist")
	ErrTagNameInvalid   = errors.New("sorry, but tag names can't look like numbers, can't contain commas and spaces")
)

type TaggableType string

const (
	TaggableTypeBookmark TaggableType = "bookmark"
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
	Subtags     []TagData
}

type GetTagOpts struct {
	GetNames   bool
	GetSubtags bool
}

type TaggableData struct {
	ID        int
	OwnerID   int
	Type      TaggableType
	CreatedAt uint64
	UpdatedAt uint64
}

type BookmarkData struct {
	// We don't embedding TaggableData here since we don't want Type to be here
	ID        int
	OwnerID   int
	CreatedAt uint64
	UpdatedAt uint64
	URL       string
	Comment   string
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
	GetTagIDByName(tx *sql.Tx, parentTagID int, tagName string) (int, error)
	GetRootTagID(tx *sql.Tx, ownerID int) (int, error)
	GetTag(tx *sql.Tx, tagID int, opts *GetTagOpts) (*TagData, error)
	GetTags(
		tx *sql.Tx, parentTagID int, opts *GetTagOpts,
	) ([]TagData, error)
	GetTagNames(tx *sql.Tx, tagID int) ([]string, error)
	//-- Taggables (bookmarks)
	CreateTaggable(tx *sql.Tx, tgbd *TaggableData) (tgbID int, err error)
	CreateBookmark(tx *sql.Tx, bd *BookmarkData) (bkmID int, err error)
}

func ValidateTagName(name string) error {
	// Tag can't look like numbers, because when we get a request which looks
	// like a number, we assume it's a tag id
	_, err := strconv.Atoi(name)
	if err == nil {
		return errors.Annotatef(ErrTagNameInvalid, "%s", name)
	}

	if strings.Contains(name, ",") ||
		strings.Contains(name, "|") || strings.Contains(name, "/") ||
		strings.Contains(name, "~") || strings.Contains(name, "=") ||
		strings.Contains(name, "'") ||
		strings.Contains(name, " ") || strings.Contains(name, "\t") ||
		!isPrintable(name) {
		return errors.Annotatef(ErrTagNameInvalid, "%s", name)
	}

	return nil
}

// checks if s is ascii and printable, aka doesn't include tab, backspace, etc.
func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}
