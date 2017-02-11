// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENCE file for details.

package postgres

import (
	"database/sql"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/storage/postgres/internal/taghier"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) GetTaggings(
	tx *sql.Tx, taggableID int, tm storage.TaggingMode,
) (tagIDs []int, err error) {
	rows, err := tx.Query("SELECT tag_id FROM taggings WHERE taggable_id = $1", taggableID)
	if err != nil {
		return nil, errors.Annotatef(
			hh.MakeInternalServerError(err),
			"getting tag ids for taggable %d", taggableID,
		)
	}
	defer rows.Close()
	for rows.Next() {
		var tagID int
		err := rows.Scan(&tagID)
		if err != nil {
			return nil, hh.MakeInternalServerError(err)
		}
		tagIDs = append(tagIDs, tagID)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Annotatef(err, "closing rows")
	}

	switch tm {
	case storage.TaggingModeAll:
		// tagIDs already contains all tag ids, return it
		return tagIDs, nil

	case storage.TaggingModeLeafs:
		// We need to return only leafs
		reg := thReg{
			s:  s,
			tx: tx,
		}
		th := taghier.New(&reg)

		for _, id := range tagIDs {
			err := th.Add(id)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}

		return th.GetLeafs(), nil

	default:
		return nil, hh.MakeInternalServerError(
			errors.Errorf("wrong tagging mode: %d", int(tm)),
		)
	}
}

func (s *StoragePostgres) SetTaggings(
	tx *sql.Tx, taggableID int, tagIDs []int, tm storage.TaggingMode,
) (err error) {
	var desired []int

	// Get desired taggings
	switch tm {
	case storage.TaggingModeAll:
		desired = tagIDs
	case storage.TaggingModeLeafs:
		reg := thReg{
			s:  s,
			tx: tx,
		}
		th := taghier.New(&reg)

		for _, id := range tagIDs {
			err := th.Add(id)
			if err != nil {
				return errors.Trace(err)
			}
		}

		desired = th.GetAll()
	}

	// Get current taggings
	current, err := s.GetTaggings(tx, taggableID, storage.TaggingModeAll)
	if err != nil {
		return errors.Trace(err)
	}

	// Calculate difference between the two
	diff := taghier.GetDiff(current, desired)

	// Apply the difference
	s.addTaggings(tx, taggableID, diff.Add)
	s.deleteTaggings(tx, taggableID, diff.Delete)

	return nil
}

func (s *StoragePostgres) addTaggings(
	tx *sql.Tx, taggableID int, tagIDsToAdd []int,
) (err error) {
	for _, tagID := range tagIDsToAdd {
		_, err := tx.Exec(
			"INSERT INTO taggings (taggable_id, tag_id) VALUES ($1, $2)",
			taggableID, tagID,
		)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

func (s *StoragePostgres) deleteTaggings(
	tx *sql.Tx, taggableID int, tagIDsToDelete []int,
) (err error) {
	for _, tagID := range tagIDsToDelete {
		_, err := tx.Exec(
			"DELETE FROM taggings WHERE taggable_id = $1 and tag_id = $2",
			taggableID, tagID,
		)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}
