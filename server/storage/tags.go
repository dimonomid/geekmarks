package storage

import (
	"database/sql"
	"path"
	"strings"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

var (
	ErrTagDoesNotExist = errors.New("tag does not exist")
)

func CreateTag(
	tx *sql.Tx, ownerID, parentTagID int, names []string,
) (tagID int, err error) {
	if len(names) == 0 {
		return 0, errors.Errorf("tag should have at least one name")
	}

	var pParent interface{}

	if parentTagID > 0 {
		// check if given parent tag id exists
		var tmpTagId int
		err := tx.QueryRow("SELECT id FROM tags WHERE id = $1", parentTagID).
			Scan(&tmpTagId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return 0, errors.Errorf("Given parent tag id %d does not exist", parentTagID)
			}
			return 0, hh.MakeInternalServerError(errors.Annotatef(
				err, "checking if parent tag id %d exists", parentTagID,
			))
		}

		pParent = parentTagID
	}

	// check if given owner exists
	{
		_, err := GetUser(tx, &GetUserArgs{ID: cptr.Int(ownerID)})
		if err != nil {
			return 0, errors.Annotatef(err, "owner id %d", ownerID)
		}
	}

	err = tx.QueryRow(
		"INSERT INTO tags (parent_id, owner_id) VALUES ($1, $2) RETURNING id",
		pParent, ownerID,
	).Scan(&tagID)
	if err != nil {
		return 0, hh.MakeInternalServerError(errors.Annotatef(
			err, "adding new tag (parent_id: %d, owner_id: %d)", pParent, ownerID,
		))
	}

	for _, name := range names {
		// Check if tag with the given name already exists under the parent tag
		// TODO: instead of calling it here manually, maybe add a SQL trigger?
		exists, err := tagExists(tx, parentTagID, name)
		if err != nil {
			return 0, errors.Trace(err)
		}
		if exists {
			return 0, errors.Errorf("Tag with the name %q already exists", name)
		}

		// Only root tag is allowed to have an empty name
		if name == "" && pParent != nil {
			return 0, errors.Errorf("Tag name can't be empty")
		}

		_, err = tx.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)",
			tagID, name,
		)
		if err != nil {
			return 0, hh.MakeInternalServerError(errors.Annotatef(
				err, "adding tag name: %q for tag with id %d", name, tagID,
			))
		}
	}

	return tagID, nil
}

func GetTagIDByPath(tx *sql.Tx, ownerID int, tagPath string) (int, error) {
	names := strings.Split(path.Clean(tagPath), "/")
	curTagID, err := GetRootTagID(tx, ownerID)
	if err != nil {
		return 0, errors.Trace(err)
	}

	for _, tagName := range names {
		if tagName == "" {
			// skip empty names
			continue
		}
		var err error
		curTagID, err = GetTagIDByName(tx, curTagID, tagName)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return curTagID, nil
}

func GetTagOwnerByID(tx *sql.Tx, tagID int) (ownerID int, err error) {
	err = tx.QueryRow(
		"SELECT owner_id FROM tags WHERE id = $1", tagID,
	).Scan(&ownerID)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return 0, interror.WrapInternalError(
				err,
				errors.Annotatef(ErrTagDoesNotExist, "%d", tagID),
			)
		}
		// Some unexpected error
		return 0, hh.MakeInternalServerError(err)
	}
	return tagID, nil
}

func GetTagIDByName(
	tx *sql.Tx, parentTagID int, tagName string,
) (int, error) {
	var tagID int
	err := tx.QueryRow(`
		SELECT t.id
			FROM tag_names n
			JOIN tags t ON n.tag_id = t.id
			WHERE t.parent_id = $1 and n.name = $2
	`, parentTagID, tagName,
	).Scan(&tagID)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return 0, interror.WrapInternalError(
				err,
				errors.Annotatef(ErrTagDoesNotExist, "%s", tagName),
			)
		}
		// Some unexpected error
		return 0, hh.MakeInternalServerError(err)
	}
	return tagID, nil
}

// GetRootTagID returns the id of the root tag for the given user.
func GetRootTagID(tx *sql.Tx, ownerID int) (int, error) {
	var rootTagID int
	err := tx.QueryRow(
		"SELECT id FROM tags WHERE owner_id = $1 AND parent_id IS NULL",
		ownerID,
	).Scan(&rootTagID)
	if err != nil {
		return 0, hh.MakeInternalServerError(
			errors.Annotatef(err, "getting root tag id for the user id %d", ownerID),
		)
	}

	return rootTagID, nil
}

// tagExists returns whether the tag with the given name already exists under
// the given parent tag.
func tagExists(tx *sql.Tx, parentTagID int, name string) (ok bool, err error) {
	var cnt int
	err = tx.QueryRow(`
		SELECT COUNT(t.id)
			FROM tag_names n
			JOIN tags t ON n.tag_id = t.id
			WHERE t.parent_id = $1 and n.name = $2
	`, parentTagID, name,
	).Scan(&cnt)
	if err != nil {
		return false, hh.MakeInternalServerError(
			errors.Annotatef(
				err,
				"checking whether tag %q already exists under the parent %d",
				name, parentTagID,
			),
		)
	}

	return cnt > 0, nil
}
