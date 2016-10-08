package postgres

import (
	"database/sql"
	"path"
	"strings"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

var ()

func (s *StoragePostgres) CreateTag(tx *sql.Tx, td *storage.TagData) (tagID int, err error) {
	if len(td.Names) == 0 {
		return 0, errors.Errorf("tag should have at least one name")
	}

	var pParent interface{}

	if td.ParentTagID > 0 {
		// check if given parent tag id exists
		var tmpTagId int
		err := tx.QueryRow("SELECT id FROM tags WHERE id = $1", td.ParentTagID).
			Scan(&tmpTagId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return 0, errors.Errorf("Given parent tag id %d does not exist", td.ParentTagID)
			}
			return 0, hh.MakeInternalServerError(errors.Annotatef(
				err, "checking if parent tag id %d exists", td.ParentTagID,
			))
		}

		pParent = td.ParentTagID
	}

	// check if given owner exists
	{
		_, err := s.GetUser(tx, &storage.GetUserArgs{ID: cptr.Int(td.OwnerID)})
		if err != nil {
			return 0, errors.Annotatef(err, "owner id %d", td.OwnerID)
		}
	}

	err = tx.QueryRow(
		"INSERT INTO tags (parent_id, owner_id, descr) VALUES ($1, $2, $3) RETURNING id",
		pParent, td.OwnerID, td.Description,
	).Scan(&tagID)
	if err != nil {
		return 0, hh.MakeInternalServerError(errors.Annotatef(
			err, "adding new tag (parent_id: %d, owner_id: %d)", pParent, td.OwnerID,
		))
	}

	for _, name := range td.Names {
		// Check if tag with the given name already exists under the parent tag
		// TODO: instead of calling it here manually, maybe add a SQL trigger?
		exists, err := s.tagExists(tx, td.ParentTagID, name)
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

func (s *StoragePostgres) GetTagIDByPath(tx *sql.Tx, ownerID int, tagPath string) (int, error) {
	names := strings.Split(path.Clean(tagPath), "/")
	curTagID, err := s.GetRootTagID(tx, ownerID)
	if err != nil {
		return 0, errors.Trace(err)
	}

	for _, tagName := range names {
		if tagName == "" {
			// skip empty names
			continue
		}
		var err error
		curTagID, err = s.GetTagIDByName(tx, curTagID, tagName)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return curTagID, nil
}

func (s *StoragePostgres) GetTagOwnerByID(tx *sql.Tx, tagID int) (ownerID int, err error) {
	err = tx.QueryRow(
		"SELECT owner_id FROM tags WHERE id = $1", tagID,
	).Scan(&ownerID)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return 0, errors.Annotatef(
				interror.WrapInternalError(
					err,
					storage.ErrTagDoesNotExist,
				),
				"%d", tagID,
			)
		}
		// Some unexpected error
		return 0, hh.MakeInternalServerError(err)
	}
	return tagID, nil
}

func (s *StoragePostgres) GetTagIDByName(
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
			return 0, errors.Annotatef(
				interror.WrapInternalError(
					err,
					storage.ErrTagDoesNotExist,
				),
				"%s", tagName,
			)
		}
		// Some unexpected error
		return 0, hh.MakeInternalServerError(err)
	}
	return tagID, nil
}

// GetRootTagID returns the id of the root tag for the given user.
func (s *StoragePostgres) GetRootTagID(tx *sql.Tx, ownerID int) (int, error) {
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
func (s *StoragePostgres) tagExists(tx *sql.Tx, parentTagID int, name string) (ok bool, err error) {
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
