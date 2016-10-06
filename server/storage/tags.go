package storage

import (
	"database/sql"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"github.com/golang/glog"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func CreateTag(
	tx *sql.Tx, ownerID, parentTagID int, names []string,
) (tagID int, err error) {
	glog.Infof("len names=%d", len(names))
	if len(names) == 0 {
		glog.Infof("returning error!")
		return 0, errors.Errorf("tag should have at least one name")
	}

	var pParent interface{}

	if parentTagID > 0 {
		pParent = parentTagID
	}

	err = tx.QueryRow(
		"INSERT INTO tags (parent_id, owner_id) VALUES ($1, $2) RETURNING id",
		pParent, ownerID,
	).Scan(&tagID)
	if err != nil {
		return 0, errors.Trace(err)
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
			return 0, errors.Trace(err)
		}
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
			err, "getting root tag id for the user",
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
			err, "checking whether tag already exists",
		)
	}

	return cnt > 0, nil
}
