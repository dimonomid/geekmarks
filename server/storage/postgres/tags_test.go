// +build all_tests integration_tests

package postgres

import (
	"database/sql"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/testutils"

	"github.com/juju/errors"
)

type tagIDs struct {
	tag1ID, tag2ID, tag3ID, tag4ID, tag5ID, tag6ID int
}

// makeTagsHierarchy creates the following tag hierarchy for the given user:
// /
// ├── tag1
// │   └── tag3
// │       ├── tag4
// │       └── tag5
// │           └── tag6
// └── tag2
func makeTagsHierarchy(tx *sql.Tx, si *StoragePostgres, ownerID int) (ids *tagIDs, err error) {
	rootTagID, err := si.GetRootTagID(tx, ownerID)
	if err != nil {
		return nil, errors.Annotatef(err, "getting root tag for user %d", ownerID)
	}

	u1Tag1ID, err := si.CreateTag(tx, &storage.TagData{
		OwnerID:     ownerID,
		ParentTagID: rootTagID,
		Description: "test tag",
		Names:       []string{"tag1", "tag1_alias"},
	})
	if err != nil {
		return nil, errors.Annotatef(err, "creating tag1 for user %d", ownerID)
	}

	u1Tag2ID, err := si.CreateTag(tx, &storage.TagData{
		OwnerID:     ownerID,
		ParentTagID: rootTagID,
		Description: "test tag",
		Names:       []string{"tag2", "tag2_alias"},
	})
	if err != nil {
		return nil, errors.Annotatef(err, "creating tag2 for user %d", ownerID)
	}

	u1Tag3ID, err := si.CreateTag(tx, &storage.TagData{
		OwnerID:     ownerID,
		ParentTagID: u1Tag1ID,
		Description: "test tag",
		Names:       []string{"tag3", "tag3_alias"},
	})
	if err != nil {
		return nil, errors.Annotatef(err, "creating tag3 for user %d", ownerID)
	}

	u1Tag4ID, err := si.CreateTag(tx, &storage.TagData{
		OwnerID:     ownerID,
		ParentTagID: u1Tag3ID,
		Description: "test tag",
		Names:       []string{"tag4", "tag4_alias"},
	})
	if err != nil {
		return nil, errors.Annotatef(err, "creating tag4 for user %d", ownerID)
	}

	u1Tag5ID, err := si.CreateTag(tx, &storage.TagData{
		OwnerID:     ownerID,
		ParentTagID: u1Tag3ID,
		Description: "test tag",
		Names:       []string{"tag5", "tag5_alias"},
	})
	if err != nil {
		return nil, errors.Annotatef(err, "creating tag5 for user %d", ownerID)
	}

	u1Tag6ID, err := si.CreateTag(tx, &storage.TagData{
		OwnerID:     ownerID,
		ParentTagID: u1Tag5ID,
		Description: "test tag",
		Names:       []string{"tag6", "tag6_alias"},
	})
	if err != nil {
		return nil, errors.Annotatef(err, "creating tag6 for user %d", ownerID)
	}

	return &tagIDs{
		tag1ID: u1Tag1ID,
		tag2ID: u1Tag2ID,
		tag3ID: u1Tag3ID,
		tag4ID: u1Tag4ID,
		tag5ID: u1Tag5ID,
		tag6ID: u1Tag6ID,
	}, nil
}

func expectPath(tx *sql.Tx, si *StoragePostgres, userID int, path string, expectedID int) error {
	tagID, err := si.GetTagIDByPath(tx, userID, path)
	if err != nil {
		return errors.Annotatef(err, "getting tag id by path %q for user %d", path, userID)
	}
	if tagID != expectedID {
		return errors.Errorf(
			"GetTagIDByPath(%d, %q) should return %d, but got %d",
			userID, path, expectedID, tagID,
		)
	}
	return nil
}

func expectPathNotFound(tx *sql.Tx, si *StoragePostgres, userID int, path string) error {
	tagID, err := si.GetTagIDByPath(tx, userID, path)
	if errors.Cause(err) != storage.ErrTagDoesNotExist {
		return errors.Errorf(
			"cause of the error returned by GetTagIDByPath(%d, %q) should be ErrTagDoesNotExist (%q), but got %q, and returned id %d",
			userID, path, storage.ErrTagDoesNotExist, errors.Cause(err), tagID,
		)
	}
	return nil
}

func TestGetTagIDByPath(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		err := si.Tx(func(tx *sql.Tx) error {
			var u1ID, u2ID int
			var err error
			if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
				return errors.Trace(err)
			}
			if u2ID, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
				return errors.Trace(err)
			}

			u1TagIDs, err := makeTagsHierarchy(tx, si, u1ID)
			if err != nil {
				return errors.Annotatef(err, "creating test tags hierarchy for user1")
			}

			u2TagIDs, err := makeTagsHierarchy(tx, si, u2ID)
			if err != nil {
				return errors.Annotatef(err, "creating test tags hierarchy for user2")
			}

			if err := expectPath(tx, si, u1ID, "/tag1/tag3/tag5/tag6", u1TagIDs.tag6ID); err != nil {
				return errors.Trace(err)
			}

			if err := expectPath(tx, si, u1ID, "tag1/tag3/tag5/tag6", u1TagIDs.tag6ID); err != nil {
				return errors.Trace(err)
			}

			if err := expectPath(tx, si, u1ID, "tag1/tag3_alias/tag5/tag6_alias", u1TagIDs.tag6ID); err != nil {
				return errors.Trace(err)
			}

			if err := expectPath(tx, si, u1ID, "/tag1/tag3/tag5", u1TagIDs.tag5ID); err != nil {
				return errors.Trace(err)
			}

			if err := expectPath(tx, si, u1ID, "/tag1/tag3/", u1TagIDs.tag3ID); err != nil {
				return errors.Trace(err)
			}

			if err := expectPath(tx, si, u1ID, "tag1", u1TagIDs.tag1ID); err != nil {
				return errors.Trace(err)
			}

			if err := expectPathNotFound(tx, si, u1ID, "/tag2/tag3"); err != nil {
				return errors.Trace(err)
			}

			if err := expectPath(tx, si, u2ID, "/tag1/tag3/tag5/tag6", u2TagIDs.tag6ID); err != nil {
				return errors.Trace(err)
			}

			return nil
		})
		return errors.Trace(err)
	})
}
