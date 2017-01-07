package storage

import (
	"database/sql"
	"strconv"
	"strings"
	"unicode"

	"github.com/juju/errors"
)

var (
	ErrUserDoesNotExist     = errors.New("user does not exist")
	ErrTagDoesNotExist      = errors.New("tag does not exist")
	ErrTagNameInvalid       = errors.New("")
	ErrBookmarkDoesNotExist = errors.New("bookmark does not exist")
)

type TaggableType string

type TagsFetchMode string
type TagNamesFetchMode string

const (
	TaggableTypeBookmark TaggableType = "bookmark"

	TagsFetchModeNone    TagsFetchMode = "none"
	TagsFetchModeLeafs   TagsFetchMode = "leafs"
	TagsFetchModeAll     TagsFetchMode = "all"
	TagsFetchModeDefault               = TagsFetchModeLeafs

	TagNamesFetchModeNone    TagNamesFetchMode = "none"
	TagNamesFetchModeShort   TagNamesFetchMode = "short"
	TagNamesFetchModeFull    TagNamesFetchMode = "full"
	TagNamesFetchModeDefault                   = TagNamesFetchModeFull
)

// TaggingMode is used for GetTaggings(), SetTaggings: specifies whether given
// argument/returned value should contain all tags (including all supertags),
// or leafs only.
type TaggingMode int

const (
	TaggingModeAll TaggingMode = iota
	TaggingModeLeafs
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
	ParentTagID *int
	Description *string
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
	// We don't embed TaggableData here since we don't want Type to be here
	ID        int
	OwnerID   int
	CreatedAt uint64
	UpdatedAt uint64
	URL       string
	Title     string
	Comment   string
}

type BookmarkDataWTags struct {
	BookmarkData
	Tags []BookmarkTag
}

type BookmarkTag struct {
	ID       int
	ParentID int
	Name     string
	FullName string
}

type TagsFetchOpts struct {
	TagsFetchMode     TagsFetchMode
	TagNamesFetchMode TagNamesFetchMode
}

type Storage interface {
	//-- Common
	Connect() error
	ApplyMigrations() error
	Tx(fn func(*sql.Tx) error) error

	//-- Users
	GetUser(tx *sql.Tx, args *GetUserArgs) (*UserData, error)
	CreateUser(tx *sql.Tx, ud *UserData) (userID int, err error)
	// Creates a new access token for a given user. If the given token is not
	// empty, use it; otherwise, a random string will be generated. In either
	// case, the effective token is returned.
	//
	// If the token is not unique, a non-nil error is returned.
	CreateAccessToken(tx *sql.Tx, userID int, token string) (string, error)
	GetUserByAccessToken(tx *sql.Tx, token string) (*UserData, error)

	//-- Tags
	CreateTag(tx *sql.Tx, td *TagData) (tagID int, err error)
	UpdateTag(tx *sql.Tx, td *TagData) (err error)
	GetTagIDByPath(tx *sql.Tx, ownerID int, tagPath string) (int, error)
	GetTagIDByName(tx *sql.Tx, parentTagID int, tagName string) (int, error)
	GetRootTagID(tx *sql.Tx, ownerID int) (int, error)
	// NOTE: all non-slice pointer fields are guaranteed to be non-nil
	GetTag(tx *sql.Tx, tagID int, opts *GetTagOpts) (*TagData, error)
	GetTags(
		tx *sql.Tx, parentTagID int, opts *GetTagOpts,
	) ([]TagData, error)
	GetTagNames(tx *sql.Tx, tagID int) ([]string, error)

	//-- Taggables (bookmarks)
	CreateTaggable(tx *sql.Tx, tgbd *TaggableData) (tgbID int, err error)
	CreateBookmark(tx *sql.Tx, bd *BookmarkData) (bkmID int, err error)
	UpdateBookmark(tx *sql.Tx, bd *BookmarkData) (err error)
	GetTaggedTaggableIDs(
		tx *sql.Tx, tagIDs []int, ownerID *int, ttypes []TaggableType,
	) (taggableIDs []int, err error)

	// tagsFetchOpts might be nil, or any of the options might be empty strings:
	// in this case, defaults will be used: TagsFetchModeLeafs and
	// TagNamesFetchModeFull.
	GetTaggedBookmarks(
		tx *sql.Tx, tagIDs []int, ownerID *int, tagsFetchOpts *TagsFetchOpts,
	) (bookmarks []BookmarkDataWTags, err error)
	GetBookmarksByURL(
		tx *sql.Tx, url string, ownerID int, tagsFetchOpts *TagsFetchOpts,
	) (bookmarks []BookmarkDataWTags, err error)
	GetBookmarkByID(
		tx *sql.Tx, bookmarkID int, tagsFetchOpts *TagsFetchOpts,
	) (bookmark *BookmarkDataWTags, err error)
	DeleteTaggable(tx *sql.Tx, taggableID int) error

	//-- Taggings
	GetTaggings(
		tx *sql.Tx, taggableID int, tm TaggingMode,
	) (tagIDs []int, err error)
	SetTaggings(
		tx *sql.Tx, taggableID int, tagIDs []int, tm TaggingMode,
	) error

	//-- Maintenance
	CheckIntegrity() error
}

func ValidateTagName(name string, allowEmpty bool) error {

	err, cleanName := CleanupTagName(name, allowEmpty)
	if err != nil {
		return errors.Trace(err)
	}

	if cleanName != name {
		return errors.Annotatef(
			// TODO: be more specific about the invalidness: elaborate on all cases
			//       when CleanupTagName might return a different name
			ErrTagNameInvalid, "%s: tag name is invalid", name,
		)
	}

	return nil
}

func CleanupTagName(s string, allowEmpty bool) (err error, name string) {
	// Tag can't look like numbers, because when we get a request which looks
	// like a number, we assume it's a tag id
	_, err = strconv.Atoi(s)
	if err == nil {
		return errors.Annotatef(
			ErrTagNameInvalid, "%s: tag name can't look like a number", s,
		), ""
	}

	hyphenAllowed := true

	for _, r := range s {
		if r == ',' || r == '|' || r == '/' || r == '~' || r == '=' || r == '\'' ||
			r == ' ' || r == '\t' || r == '\n' || r == '\r' || !unicode.IsPrint(r) {
			r = '-'
		}

		if r == '-' {
			if hyphenAllowed {
				name += string(r)
				hyphenAllowed = false
			}
		} else {
			name += string(r)
			hyphenAllowed = true
		}
	}

	// Tag can't start and end with a hyphen
	name = strings.Trim(name, "-")

	if !allowEmpty && name == "" {
		return errors.Annotatef(ErrTagNameInvalid, "tag name can't be empty"), ""
	}

	return nil, name
}
