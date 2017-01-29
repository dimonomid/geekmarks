package postgres

import (
	"database/sql"
	"fmt"

	"dmitryfrank.com/geekmarks/server/storage/postgres/internal/taghier"

	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

type childrenCheck struct {
	id                int
	childrenCnt       int
	childrenCntActual int
}

func (cc *childrenCheck) String() string {
	return fmt.Sprintf("id=%d, childrenCnt=%d, childrenCntActual=%d",
		cc.id, cc.childrenCnt, cc.childrenCntActual,
	)
}

func (s *StoragePostgres) CheckIntegrity() error {
	err := s.TxOpt(TxILevelRepeatableRead, TxModeReadOnly, func(tx *sql.Tx) error {
		err := s.checkChildrenCnt(tx)
		if err != nil {
			return errors.Trace(err)
		}

		err = s.checkTaggings(tx)
		if err != nil {
			return errors.Trace(err)
		}

		err = s.checkOnlyRootTagging(tx)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (s *StoragePostgres) checkTaggings(tx *sql.Tx) error {

	// Get all users, for each of them:
	// - Get all user's tags
	// - Feed all of them to taghier
	// - Make sure that the taghier contains just a single root
	// - For each of the tag, theck that all taggings contain the full path to
	//   the tag

	// Check all users
	users, err := s.GetUsers(tx)
	if err != nil {
		return errors.Trace(err)
	}

	for _, user := range users {
		reg := thReg{
			s:  s,
			tx: tx,
		}
		th := taghier.New(&reg)

		// Get all user's tags and feed them to the taghier
		rows, err := tx.Query("SELECT id FROM tags WHERE owner_id = $1", user.ID)
		if err != nil {
			return errors.Trace(err)
		}

		var tagIDs []int

		defer rows.Close()
		for rows.Next() {
			var curID int

			err := rows.Scan(&curID)
			if err != nil {
				return errors.Trace(err)
			}

			// NOTE: we can't call th.Add() right here, because it will also hit a
			// database, and iterating through more than one Rows at a time is an
			// error
			tagIDs = append(tagIDs, curID)
		}

		for _, curID := range tagIDs {
			th.Add(curID)
		}

		// Now, th contains all tags for the current user

		// Make sure taghier contains just a single root
		roots := th.GetRoots()
		if rootsCnt := len(roots); rootsCnt != 1 {
			return errors.Errorf(
				"user %d: tag roots count is %d (should be 1). tag roots: %q",
				user.ID, rootsCnt, roots,
			)
		}

		// For each of the tags, make sure that there is a tagging for each
		// tag in the current tag's path
		for _, tag := range th.GetAll() {
			path := th.GetPath(tag)

			// This commented code intentionally breaks integrity, so that I can be
			// sure that checkFullTaggingsPath works. I'm leaving it here for now.

			//if len(path) >= 4 {
			//_, err := tx.Exec(
			//"DELETE FROM taggings WHERE tag_id = $1", path[2],
			//)
			//if err != nil {
			//return errors.Trace(err)
			//}
			//}

			if err := s.checkFullTaggingsPath(tx, path); err != nil {
				return errors.Annotatef(err, "user %d", user.ID)
			}
		}
	}

	return nil
}

func (s *StoragePostgres) checkFullTaggingsPath(tx *sql.Tx, path []int) error {
	var taggableIDs []int

	// If there is less than 2 items in the path, there's no need to check
	// anything
	if len(path) < 2 {
		return nil
	}

	// Reverse path so that it goes from the leaf to the root
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	// Build query which finds taggable_ids which are tagged with path[0], but
	// not tagged with any of the other path items.
	//
	// E.g. if path is {300, 200, 100}, the query will be:
	//
	// SELECT t.taggable_id FROM taggings t
	//   FULL OUTER JOIN taggings t1 ON
	//     (t1.taggable_id=t.taggable_id AND t1.tag_id=200)
	//   FULL OUTER JOIN taggings t2 ON
	//     (t2.taggable_id=t.taggable_id AND t2.tag_id=100)
	//   WHERE t.tag_id=300 AND (t1.taggable_id IS NULL OR t2.taggable_id IS NULL)
	//
	// So if the result is not empty, it means that the integrity is broken.
	phNum := 1
	args := []interface{}{}
	query := "SELECT t.taggable_id FROM taggings t "

	subq := ""

	for k, tagID := range path {
		// First tag is a leaf, its id will be in a WHERE clause later
		if k == 0 {
			continue
		}

		query += fmt.Sprintf(` FULL OUTER JOIN taggings t%d
        ON (t%d.taggable_id=t.taggable_id AND t%d.tag_id=$%d) `,
			k, k, k,
			phNum,
		)
		phNum++

		if subq != "" {
			subq += " OR "
		}
		subq += fmt.Sprintf("t%d.taggable_id IS NULL", k)

		args = append(args, tagID)
	}

	query += fmt.Sprintf("WHERE t.tag_id=$%d AND (%s)", phNum, subq)
	phNum++
	args = append(args, path[0])

	// Execute it
	rows, err := tx.Query(query, args...)
	if err != nil {
		return errors.Trace(err)
	}

	defer rows.Close()
	for rows.Next() {
		var taggableID int
		err := rows.Scan(&taggableID)
		if err != nil {
			return errors.Trace(err)
		}
		taggableIDs = append(taggableIDs, taggableID)
	}
	if err := rows.Close(); err != nil {
		return errors.Annotatef(err, "closing rows")
	}

	if len(taggableIDs) > 0 {
		return errors.Errorf(
			"for the tag %d (full path: %v) some intermediate taggings are missing for the following tags: %v",
			path[0], path, taggableIDs,
		)
	}

	return nil
}

// It's illegal for the taggable to be tagged with the root tag only,
// so checkOnlyRootTagging checks for these cases
func (s *StoragePostgres) checkOnlyRootTagging(tx *sql.Tx) error {
	// Get all root tags
	rootTagIDs := []int{}
	rows, err := tx.Query("SELECT id FROM tags WHERE parent_id IS NULL")
	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()
	for rows.Next() {
		var cur int
		err := rows.Scan(&cur)
		if err != nil {
			return errors.Trace(err)
		}
		rootTagIDs = append(rootTagIDs, cur)
	}
	rows.Close()

	// For each of the root tags, make sure there's no taggables tagged only
	// with this one tag
	for _, rootTagID := range rootTagIDs {
		badIDs, err := s.getTaggablesTaggedWithOnlyOneTag(tx, rootTagID)
		if err != nil {
			return errors.Trace(err)
		}
		if len(badIDs) > 0 {
			return errors.Errorf(
				"some taggables (ids: %v) are tagged with root tag only (id: %d), this is illegal",
				badIDs, rootTagID,
			)
		}
	}

	return nil
}

func (s *StoragePostgres) checkChildrenCnt(tx *sql.Tx) error {
	rows, err := tx.Query(`
SELECT id, children_cnt, children_cnt_actual
  FROM
    (SELECT id, children_cnt,
            (SELECT COUNT(id) FROM tags WHERE parent_id = t.id) AS children_cnt_actual
          FROM tags t) T
  WHERE children_cnt != children_cnt_actual
`,
	)
	if err != nil {
		return errors.Trace(err)
	}

	failures := []childrenCheck{}
	str := ""

	defer rows.Close()
	for rows.Next() {
		var cur childrenCheck
		err := rows.Scan(&cur.id, &cur.childrenCnt, &cur.childrenCntActual)
		if err != nil {
			return errors.Trace(err)
		}

		failures = append(failures, cur)
		str += cur.String() + "\n"
	}

	if len(str) > 0 {
		return errors.Errorf("children count integrity is broken: %s", str)
	}

	return nil
}
