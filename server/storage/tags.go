package storage

import (
	"database/sql"

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
		// TODO: ensure that the tag with this name does not already exists
		// (in the given parent tag and owned by the same user, of course).
		// Either do that in Go or, if possible, in SQL

		// Only root tag is allowed to have an empty name
		if name == "" && pParent != nil {
			return 0, errors.Errorf("Tag name can't be empty")
		}

		_, err := tx.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)",
			tagID, name,
		)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return tagID, nil
}

func GetRootTagID(tx *sql.Tx, ownerID int) (int, error) {
	var rootTagID int
	err := tx.QueryRow(
		"SELECT id FROM tags WHERE owner_id = $1 AND parent_id IS NULL",
		ownerID,
	).Scan(&rootTagID)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return rootTagID, nil
}
