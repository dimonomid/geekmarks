// +build all_tests integration_tests

package postgres

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"dmitryfrank.com/geekmarks/server/testutils"

	"github.com/juju/errors"
)

func TestTagWithInvalidOwner(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		_, err := si.db.Exec(
			"INSERT INTO tags (parent_id, owner_id) VALUES (NULL, 100)",
		)
		if err == nil {
			return errors.Errorf("should be error")
		}
		if !strings.Contains(err.Error(), "foreign") {
			return errors.Errorf("error should contain \"foreign\", but it doesn't: %q", err)
		}
		return nil
	})
}

func TestTagWithNullOwner(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		_, err := si.db.Exec(
			"INSERT INTO tags (parent_id, owner_id) VALUES (NULL, NULL)",
		)
		if err == nil {
			return errors.Errorf("should be error")
		}
		if !strings.Contains(err.Error(), "not-null") {
			return errors.Errorf("error should contain \"not-null\", but it doesn't: %q", err)
		}
		return nil
	})
}

func TestTagWithWrongParent(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {

		var u1ID int
		var err error
		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}

		_, err = si.db.Exec(
			"INSERT INTO tags (parent_id, owner_id) VALUES (100, $1)", u1ID,
		)
		if err == nil {
			return errors.Errorf("should be error")
		}
		if !strings.Contains(err.Error(), "foreign") {
			return errors.Errorf("error should contain \"foreign\", but it doesn't: %q", err)
		}

		return nil
	})
}

func TestCreationOfDuplicateTagName(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {

		var u1ID int
		var err error
		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}

		var u1RootTagID int
		err = si.Tx(func(tx *sql.Tx) error {
			var err error
			u1RootTagID, err = si.GetRootTagID(tx, u1ID)
			if err != nil {
				return errors.Annotatef(err, "getting root tag for the user %d", u1ID)
			}
			return nil
		})
		if err != nil {
			return errors.Trace(err)
		}

		var u1Tag1ID int

		err = si.db.QueryRow(
			"INSERT INTO tags (parent_id, owner_id) VALUES ($1, $2) RETURNING id", u1RootTagID, u1ID,
		).Scan(&u1Tag1ID)
		if err != nil {
			return errors.Annotatef(err, "creating tag1 for user %d", u1ID)
		}

		// Create two equal names for the tag {{{
		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u1Tag1ID, "tag1",
		)
		if err != nil {
			return errors.Annotatef(err, "creating tag name for tag %d", u1Tag1ID)
		}

		// Create record with the same name: should fail
		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u1Tag1ID, "tag1",
		)
		if err == nil {
			return errors.Errorf("should be error: violation of unique constraint")
		}
		if !strings.Contains(err.Error(), "tag_names_pkey") {
			return errors.Errorf("error should contain \"tag_names_pkey\", but it doesn't: %q", err)
		}
		// }}}

		return nil
	})
}

func TestDoubleRootTags(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		var u1ID int
		var err error
		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}

		var u1RootTagID int
		err = si.db.QueryRow(
			"INSERT INTO tags (parent_id, owner_id) VALUES (NULL, $1) RETURNING id", u1ID,
		).Scan(&u1RootTagID)
		if err == nil {
			return errors.Errorf(
				"inserting second tag with NULL parent should result in an error, but instead a record with id %d was created",
				u1RootTagID,
			)
		}

		return nil
	})
}

func TestOnDeleteCascade(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {

		var u1ID, u2ID int
		var err error
		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		if u2ID, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}

		// Create root tags for both users
		var u1RootTagID, u2RootTagID int
		err = si.Tx(func(tx *sql.Tx) error {
			var err error
			u1RootTagID, err = si.GetRootTagID(tx, u1ID)
			if err != nil {
				return errors.Annotatef(err, "getting root tag for the user %d", u1ID)
			}
			u2RootTagID, err = si.GetRootTagID(tx, u2ID)
			if err != nil {
				return errors.Annotatef(err, "getting root tag for the user %d", u2ID)
			}
			return nil
		})
		if err != nil {
			return errors.Trace(err)
		}

		var u1Tag1ID, u1Tag2ID, u2Tag1ID int

		// Create tag1 for both users
		// Create tag1 for user1 {{{
		err = si.db.QueryRow(
			"INSERT INTO tags (parent_id, owner_id) VALUES ($1, $2) RETURNING id", u1RootTagID, u1ID,
		).Scan(&u1Tag1ID)
		if err != nil {
			return errors.Annotatef(err, "creating tag1 for user %d", u1ID)
		}

		// Create two names for the tag {{{
		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u1Tag1ID, "tag1",
		)
		if err != nil {
			return errors.Annotatef(err, "creating tag name for tag %d", u1Tag1ID)
		}

		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u1Tag1ID, "tag1_alias",
		)
		if err != nil {
			return errors.Annotatef(err, "creating tag name for tag %d", u1Tag1ID)
		}
		// }}}
		// }}}

		// Create tag2 for user1 {{{
		err = si.db.QueryRow(
			"INSERT INTO tags (parent_id, owner_id) VALUES ($1, $2) RETURNING id", u1RootTagID, u1ID,
		).Scan(&u1Tag2ID)
		if err != nil {
			return errors.Annotatef(err, "creating tag2 for user %d", u1ID)
		}

		// Create two names for the tag {{{
		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u1Tag2ID, "tag2",
		)
		if err != nil {
			return errors.Annotatef(err, "creating tag name for tag %d", u1Tag2ID)
		}

		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u1Tag2ID, "tag2_alias",
		)
		if err != nil {
			return errors.Annotatef(err, "creating tag name for tag %d", u1Tag2ID)
		}
		// }}}
		// }}}

		// Create tag1 for user2 {{{
		err = si.db.QueryRow(
			"INSERT INTO tags (parent_id, owner_id) VALUES ($1, $2) RETURNING id", u2RootTagID, u2ID,
		).Scan(&u2Tag1ID)
		if err != nil {
			return errors.Annotatef(err, "creating tag1 for user %d", u2ID)
		}

		// Create two names for the tag {{{
		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u2Tag1ID, "tag1",
		)
		if err != nil {
			return errors.Annotatef(err, "creating tag name for tag %d", u2Tag1ID)
		}

		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u2Tag1ID, "tag1_alias",
		)
		if err != nil {
			return errors.Annotatef(err, "creating tag name for tag %d", u2Tag1ID)
		}
		// }}}
		// }}}

		var tagNamesCntAll, tagNamesCntNoUser1 int

		// Try to all tags names
		err = si.db.QueryRow(
			"SELECT COUNT(*) FROM tag_names",
		).Scan(&tagNamesCntAll)
		if err != nil {
			return errors.Annotatef(err, "selecting count(*) for tag_names")
		}

		// Delete user1
		_, err = si.db.Exec(
			"DELETE FROM users WHERE id = $1", u1ID,
		)
		if err != nil {
			return errors.Annotatef(err, "deleting test user 1 (id %d)", u1ID)
		}

		// Try to all tags names
		err = si.db.QueryRow(
			"SELECT COUNT(*) FROM tag_names",
		).Scan(&tagNamesCntNoUser1)
		if err != nil {
			return errors.Annotatef(err, "selecting count(*) for tag_names")
		}
		if tagNamesCntNoUser1 >= tagNamesCntAll {
			return errors.Errorf("tagNamesCntNoUser1 should be < tagNamesCntAll")
		}

		var cnt int

		// Try to get user1's tags: should be 0
		err = si.db.QueryRow(
			"SELECT COUNT(id) FROM tags WHERE owner_id = $1", u1ID,
		).Scan(&cnt)
		if err != nil {
			return errors.Annotatef(err, "selecting count(id) for user id %d", u1ID)
		}
		if cnt != 0 {
			return errors.Errorf("tags cnt should be 0, but it is %d", cnt)
		}

		// Try to get user2's tags: should be > 0
		err = si.db.QueryRow(
			"SELECT COUNT(id) FROM tags WHERE owner_id = $1", u2ID,
		).Scan(&cnt)
		if err != nil {
			return errors.Annotatef(err, "selecting count(id) for user id %d", u2ID)
		}
		if cnt == 0 {
			return errors.Errorf("tags cnt should not be 0, but it is %d", cnt)
		}

		// Delete user2
		_, err = si.db.Exec(
			"DELETE FROM users WHERE id = $1", u2ID,
		)
		if err != nil {
			return errors.Annotatef(err, "deleting test user 2 (id %d)", u2ID)
		}

		// Try to all tags: should be 0
		err = si.db.QueryRow(
			"SELECT COUNT(id) FROM tags",
		).Scan(&cnt)
		if err != nil {
			return errors.Annotatef(err, "selecting count(id) for tags")
		}
		if cnt != 0 {
			return errors.Errorf("tags cnt should be 0, but it is %d", cnt)
		}

		// Try to all tags names: should be 0
		err = si.db.QueryRow(
			"SELECT COUNT(*) FROM tag_names",
		).Scan(&cnt)
		if err != nil {
			return errors.Annotatef(err, "selecting count(*) for tag_names")
		}
		if cnt != 0 {
			return errors.Errorf("tag_names cnt should be 0, but it is %d", cnt)
		}

		return nil
	})
}

func TestTaggablesTimes(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {

		var u1ID int
		var tgb1ID, tgb2ID int
		var err error
		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}

		time1 := time.Now()
		err = si.db.QueryRow(
			"INSERT INTO taggables (owner_id, type) VALUES ($1, $2) RETURNING id",
			u1ID, "bookmark",
		).Scan(&tgb1ID)
		if err != nil {
			return errors.Trace(err)
		}

		time.Sleep(time.Millisecond * 100)

		time2 := time.Now()
		err = si.db.QueryRow(
			"INSERT INTO taggables (owner_id, type) VALUES ($1, $2) RETURNING id",
			u1ID, "bookmark",
		).Scan(&tgb2ID)
		if err != nil {
			return errors.Trace(err)
		}

		// Check that tgb1 has correct times
		var tgb1createdAt, tgb1updatedAt float64
		err = si.db.QueryRow(
			`SELECT
			extract(epoch from created_ts) as c,
			extract(epoch from updated_ts) as u
			FROM taggables WHERE id = $1`,
			tgb1ID,
		).Scan(&tgb1createdAt, &tgb1updatedAt)
		if err != nil {
			return errors.Trace(err)
		}

		if err := compareTime(t, time1, tgb1createdAt); err != nil {
			return errors.Annotatef(err, "created_ts")
		}

		if tgb1createdAt != tgb1updatedAt {
			return errors.Errorf("created and updated time should be equal")
		}

		// Check that tgb2 has correct times

		var tgb2createdAt, tgb2updatedAt float64
		err = si.db.QueryRow(
			`SELECT
			extract(epoch from created_ts) as c,
			extract(epoch from updated_ts) as u
			FROM taggables WHERE id = $1`,
			tgb2ID,
		).Scan(&tgb2createdAt, &tgb2updatedAt)
		if err != nil {
			return errors.Trace(err)
		}

		if err := compareTime(t, time2, tgb2createdAt); err != nil {
			return errors.Annotatef(err, "created_ts")
		}

		if tgb2createdAt != tgb2updatedAt {
			return errors.Errorf("created and updated time should be equal")
		}

		if tgb1createdAt == tgb2createdAt {
			return errors.Errorf("creation time for two taggables should not be equal")
		}

		// Update tgb1

		time1upd := time.Now()
		_, err = si.db.Exec(
			"UPDATE taggables SET id = id WHERE id = $1", tgb1ID,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Check that tgb1 has correct times
		var tgb1createdAt2, tgb1updatedAt2 float64
		err = si.db.QueryRow(
			`SELECT
			extract(epoch from created_ts) as c,
			extract(epoch from updated_ts) as u
			FROM taggables WHERE id = $1`,
			tgb1ID,
		).Scan(&tgb1createdAt2, &tgb1updatedAt2)
		if err != nil {
			return errors.Trace(err)
		}

		if err := compareTime(t, time1upd, tgb1createdAt2); err != nil {
			return errors.Annotatef(err, "created_ts")
		}

		if tgb1createdAt2 == tgb1updatedAt2 {
			return errors.Errorf("created and updated time should NOT be equal now")
		}

		if tgb1createdAt2 != tgb1createdAt {
			return errors.Errorf("creation time should not change")
		}

		return nil
	})
}

func compareTime(t *testing.T, expected time.Time, got float64) error {
	exp := float64(expected.Unix())
	if got < exp-1 || got > exp+1 {
		return errors.Errorf("expected: %f, got %f", exp, got)
	}
	return nil
}
