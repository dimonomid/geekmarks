// +build all_tests integration_tests

package postgres

import (
	"flag"
	"os"
	"strings"
	"testing"

	"dmitryfrank.com/geekmarks/server/testutils"

	"github.com/juju/errors"
)

var (
	postgresURL = flag.String("geekmarks.postgres.url", "",
		"Data source name pointing to the Postgres database. Alternatively, can be "+
			"given in an environment variable GM_POSTGRES_URL.")
)

func runWithRealDB(t *testing.T, f func(si *StoragePostgres) error) {
	pgURL := *postgresURL
	if pgURL == "" {
		pgURL = os.Getenv("GM_POSTGRES_URL")
	}
	si, err := New(pgURL)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = si.Connect()
	if err != nil {
		t.Errorf("%s", err)
	}

	err = testutils.PrepareTestDB(t, si)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = f(si)
	if err != nil {
		t.Errorf("%s", err)
	}

	err = testutils.CleanupTestDB(t)
	if err != nil {
		t.Errorf("%s", err)
	}
}

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

		// Create root tag for user1 {{{
		err = si.db.QueryRow(
			"INSERT INTO tags (parent_id, owner_id) VALUES (NULL, $1) RETURNING id", u1ID,
		).Scan(&u1RootTagID)
		if err != nil {
			return errors.Annotatef(err, "creating root tag for user %d", u1ID)
		}

		// Create empty name for the root tag {{{
		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u1RootTagID, "",
		)
		if err != nil {
			return errors.Annotatef(err, "creating name for tag %d", u1RootTagID)
		}
		// }}}
		// }}}

		// Create root tag for user2 {{{
		err = si.db.QueryRow(
			"INSERT INTO tags (parent_id, owner_id) VALUES (NULL, $1) RETURNING id", u2ID,
		).Scan(&u2RootTagID)
		if err != nil {
			return errors.Annotatef(err, "creating root tag for user %d", u2ID)
		}

		// Create empty name for the root tag {{{
		_, err = si.db.Exec(
			"INSERT INTO tag_names (tag_id, name) VALUES ($1, $2)", u2RootTagID, "",
		)
		if err != nil {
			return errors.Annotatef(err, "creating name for tag %d", u2RootTagID)
		}
		// }}}
		// }}}

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
			"SELECT COUNT(id) FROM tag_names",
		).Scan(&tagNamesCntAll)
		if err != nil {
			return errors.Annotatef(err, "selecting count(id) for tag_names")
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
			"SELECT COUNT(id) FROM tag_names",
		).Scan(&tagNamesCntNoUser1)
		if err != nil {
			return errors.Annotatef(err, "selecting count(id) for tag_names")
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
			"SELECT COUNT(id) FROM tag_names",
		).Scan(&cnt)
		if err != nil {
			return errors.Annotatef(err, "selecting count(id) for tag_names")
		}
		if cnt != 0 {
			return errors.Errorf("tag_names cnt should be 0, but it is %d", cnt)
		}

		return nil
	})
}
