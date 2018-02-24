// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package postgres

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"github.com/dimonomid/interrors"
	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/storage/postgres/internal/taghier"
	"github.com/golang/glog"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) CreateTag(
	tx *sql.Tx, td *storage.TagData,
) (tagID int, err error) {
	if len(td.Names) == 0 {
		return 0, errors.Errorf("tag should have at least one name")
	}

	var iParentID interface{}
	var parentID int

	if td.ParentTagID != nil {
		parentID = *td.ParentTagID
	}

	if parentID > 0 {
		// check if given parent tag id exists
		var tmpTagId int
		err := tx.QueryRow("SELECT id FROM tags WHERE id = $1", parentID).
			Scan(&tmpTagId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return 0, errors.Errorf("Given parent tag id %d does not exist", parentID)
			}
			return 0, hh.MakeInternalServerError(errors.Annotatef(
				err, "checking if parent tag id %d exists", parentID,
			))
		}

		// increment children count of the parent
		_, err = tx.Exec("UPDATE tags SET children_cnt = children_cnt + 1 WHERE id = $1", parentID)
		if err != nil {
			return 0, hh.MakeInternalServerError(errors.Annotatef(
				err, "incrementing children_cnt of the tag with id %d", parentID,
			))
		}

		iParentID = parentID
	}

	// check if given owner exists
	{
		_, err := s.GetUser(tx, &storage.GetUserArgs{ID: cptr.Int(td.OwnerID)})
		if err != nil {
			return 0, errors.Annotatef(err, "owner id %d", td.OwnerID)
		}
	}

	description := ""
	if td.Description != nil {
		description = *td.Description
	}

	err = tx.QueryRow(
		"INSERT INTO tags (parent_id, owner_id, descr) VALUES ($1, $2, $3) RETURNING id",
		iParentID, td.OwnerID, description,
	).Scan(&tagID)
	if err != nil {
		return 0, hh.MakeInternalServerError(errors.Annotatef(
			err, "adding new tag (parent_id: %d, owner_id: %d)", iParentID, td.OwnerID,
		))
	}

	// Add all names
	for i, name := range td.Names {
		if err := s.addTagName(
			tx, tagID, parentID, name,
			(i == 0),           // primary
			(iParentID == nil), // allowEmpty
		); err != nil {
			return 0, errors.Trace(err)
		}
	}

	// Create all subtags
	for _, subTag := range td.Subtags {
		_, err := s.CreateTag(tx, &subTag)
		if err != nil {
			return 0, errors.Annotatef(err, "creating subtag")
		}
	}

	return tagID, nil
}

func (s *StoragePostgres) UpdateTag(
	tx *sql.Tx, td *storage.TagData, leafPolicy storage.TaggableLeafPolicy,
) (err error) {
	// Move tag, if needed {{{
	if td.ParentTagID != nil {
		// We need to move the tag under another tag

		reg := thReg{
			s:  s,
			tx: tx,
		}
		hierProto := taghier.New(&reg)

		if err := hierProto.Add(td.ID); err != nil {
			return errors.Trace(err)
		}

		if err := hierProto.Add(*td.ParentTagID); err != nil {
			return errors.Trace(err)
		}

		// Make sure that the new parent is not the current tag or one of its
		// descendants
		isSubnode, err := hierProto.IsSubnode(*td.ParentTagID, td.ID)
		if err != nil {
			return errors.Trace(err)
		}

		if *td.ParentTagID == td.ID || isSubnode {
			return errors.Errorf("tag cannot be moved under itself or one of its descendants")
		}

		oldParentID := hierProto.GetParent(td.ID)

		fmt.Printf("oldparent=%d\n", oldParentID)

		// Get affected bookmarks (those tagged with the original tag id and its
		// descendants)
		taggableIDs, err := s.GetTaggedTaggableIDs(tx, []int{td.ID}, nil, nil)
		if err != nil {
			return errors.Trace(err)
		}

		fmt.Printf("affected taggable ids: %v\n", taggableIDs)

		// For all the affected bookmarks, calculate the difference and apply
		for _, taggableID := range taggableIDs {
			// Get all taggings for the current bookmark
			tagIDs, err := s.GetTaggings(tx, taggableID, storage.TaggingModeAll)
			if err != nil {
				return errors.Trace(err)
			}

			// Create a warmed-up copy of taghier instance, and feed all current tag
			// IDs to it
			hierCur := hierProto.MakeCopy()
			for _, id := range tagIDs {
				if err := hierCur.Add(id); err != nil {
					return errors.Trace(err)
				}
			}

			var removeNewLeafs bool
			switch leafPolicy {
			case storage.TaggableLeafPolicyKeep:
				removeNewLeafs = false
			case storage.TaggableLeafPolicyDel:
				removeNewLeafs = true
			default:
				return errors.Errorf("invalid leafPolicy: %q", leafPolicy)
			}

			// Perform the in-memory move, and delete all new leafs
			if err := hierCur.Move(td.ID, *td.ParentTagID, removeNewLeafs); err != nil {
				return errors.Trace(err)
			}

			// Apply the taggings change
			err = s.SetTaggings(tx, taggableID, hierCur.GetAll(), storage.TaggingModeAll)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// Update parent_id of the moved tag
		_, err = tx.Exec(
			"UPDATE tags SET parent_id = $1 WHERE id = $2", *td.ParentTagID, td.ID,
		)
		if err != nil {
			return hh.MakeInternalServerError(errors.Annotatef(
				err, "updating tag parent_id (id: %d, parent_id: %q)",
				td.ID, *td.ParentTagID,
			))
		}

		// Update childrent_cnt of the two parents
		_, err = tx.Exec(
			"UPDATE tags SET children_cnt = children_cnt - 1 WHERE id = $1", oldParentID,
		)
		if err != nil {
			return hh.MakeInternalServerError(errors.Annotatef(
				err, "decrementing children_cnt of the tag %d", oldParentID,
			))
		}

		_, err = tx.Exec(
			"UPDATE tags SET children_cnt = children_cnt + 1 WHERE id = $1", *td.ParentTagID,
		)
		if err != nil {
			return hh.MakeInternalServerError(errors.Annotatef(
				err, "incrementing children_cnt of the tag %d", *td.ParentTagID,
			))
		}
	}
	// }}}

	// Update tag description, if needed {{{
	if td.Description != nil {
		_, err = tx.Exec(
			"UPDATE tags SET descr = $1 WHERE id = $2", td.Description, td.ID,
		)
		if err != nil {
			return hh.MakeInternalServerError(errors.Annotatef(
				err, "updating tag description (id: %d, description: %q)",
				td.ID, td.Description,
			))
		}
	}
	// }}}

	// Update tag names, if needed {{{
	if td.Names != nil {
		if len(td.Names) == 0 {
			return errors.Errorf("tag should have at least one name")
		}

		curNames, err := s.GetTagNames(tx, td.ID)
		if err != nil {
			return errors.Trace(err)
		}

		namesDiff := s.getNamesDiff(curNames, td.Names)

		// Apply the names difference
		if len(namesDiff.add) > 0 {
			// To add a name, we need to know a tag parent's ID (it's used for the
			// check whether a tag with the given name already exists under the parent)
			existingTD, err := s.GetTag(tx, td.ID, &storage.GetTagOpts{})
			if err != nil {
				return errors.Trace(err)
			}
			tagParentID := *existingTD.ParentTagID

			for _, name := range namesDiff.add {
				if err := s.addTagName(
					tx, td.ID, tagParentID, name,
					false, // not primary (primary name will be adjusted later, if needed)
					false, // do not allow empty
				); err != nil {
					return errors.Trace(err)
				}
			}
		}

		for _, name := range namesDiff.delete {
			if err := s.deleteTagName(tx, td.ID, name); err != nil {
				return errors.Trace(err)
			}
		}

		// If needed, adjust primary name
		if namesDiff.clearPrimary != nil {
			s.setTagNamePrimary(tx, td.ID, *namesDiff.clearPrimary, false)
		}
		if namesDiff.setPrimary != nil {
			s.setTagNamePrimary(tx, td.ID, *namesDiff.setPrimary, true)
		}
	}
	// }}}

	return nil
}

func (s *StoragePostgres) DeleteTag(
	tx *sql.Tx, tagID int, leafPolicy storage.TaggableLeafPolicy,
) (err error) {
	// TODO: so far only "keep new leaf" policy is implemented for tag deletion
	if leafPolicy != storage.TaggableLeafPolicyKeep {
		return errors.Annotatef(
			storage.ErrNotImplemented,
			"so far, only \"keep new leaf\" policy is implemented",
		)
	}

	td, err := s.GetTag(tx, tagID, &storage.GetTagOpts{})
	if err != nil {
		return errors.Trace(err)
	}

	// Make sure the tag to be deleted is not the user's root tag
	rootTagID, err := s.GetRootTagID(tx, td.OwnerID)
	if err != nil {
		return errors.Trace(err)
	}
	if tagID == rootTagID {
		glog.V(2).Infof("tried to delete the root tag")
		return errors.Errorf("cowardly refused to delete the root tag")
	}

	// Here we just delete the subject tag; all the subtags and taggings
	// will be deleted automatically thanks to ON DELETE CASCADE
	_, err = tx.Exec("DELETE FROM tags WHERE id = $1", tagID)
	if err != nil {
		return hh.MakeInternalServerError(errors.Annotatef(
			err, "deleting the tag with id %d", tagID,
		))
	}

	_, err = tx.Exec("UPDATE tags SET children_cnt = children_cnt - 1 WHERE id = $1", td.ParentTagID)
	if err != nil {
		return hh.MakeInternalServerError(errors.Annotatef(
			err, "decrementing children_cnt of the tag with id %d", td.ParentTagID,
		))
	}

	// if ParentTagID is a root tag for the user, then we should find
	// bookmarks tagged with only this flag, and make them untagged
	// (remove tagging by the root tag)
	if *td.ParentTagID == rootTagID {
		tgbIDs, err := s.getTaggablesTaggedWithOnlyOneTag(tx, rootTagID)
		if err != nil {
			return errors.Trace(err)
		}
		for _, curTgbID := range tgbIDs {
			err := s.SetTaggings(tx, curTgbID, []int{}, storage.TaggingModeAll)
			if err != nil {
				return errors.Trace(err)
			}
		}
	}

	return err
}

func (s *StoragePostgres) GetTagIDByPath(tx *sql.Tx, ownerID int, tagPath string) (int, error) {
	names := strings.Split(tagPath, "/")
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
				interrors.WrapInternalError(
					err,
					storage.ErrTagDoesNotExist,
				),
				"%q", tagName,
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

func (s *StoragePostgres) GetTagNames(tx *sql.Tx, tagID int) ([]string, error) {
	var tagNames []string
	rows, err := tx.Query(`SELECT name FROM tag_names WHERE tag_id = $1 ORDER BY "primary" DESC`, tagID)
	if err != nil {
		return nil, errors.Annotatef(
			hh.MakeInternalServerError(err),
			"getting tag names for tag %d", tagID,
		)
	}
	defer rows.Close()
	for rows.Next() {
		var tagName string
		err := rows.Scan(&tagName)
		if err != nil {
			return nil, hh.MakeInternalServerError(err)
		}

		tagNames = append(tagNames, tagName)
	}

	if err := rows.Close(); err != nil {
		return nil, errors.Annotatef(err, "closing rows")
	}

	return tagNames, nil
}

func (s *StoragePostgres) GetTag(
	tx *sql.Tx, tagID int, opts *storage.GetTagOpts,
) (*storage.TagData, error) {
	tagsData, err := s.getTagsInternal(tx, "id", tagID, opts)
	if err != nil {
		return nil, errors.Trace(err)
	}

	if len(tagsData) == 0 {
		return nil, storage.ErrTagDoesNotExist
	}

	if len(tagsData) > 1 {
		return nil, hh.MakeInternalServerError(
			errors.Errorf("getTagsInternal() should have returned just 1 row, but it returned %d", len(tagsData)),
		)
	}

	return &tagsData[0], nil
}

func (s *StoragePostgres) GetTags(
	tx *sql.Tx, parentTagID int, opts *storage.GetTagOpts,
) ([]storage.TagData, error) {
	tagsData, err := s.getTagsInternal(tx, "parent_id", parentTagID, opts)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return tagsData, nil
}

func (s *StoragePostgres) getTagsInternal(
	tx *sql.Tx, fieldName string, tagID int, opts *storage.GetTagOpts,
) ([]storage.TagData, error) {
	var tagsData []storage.TagData
	var childrenCntArr []int
	if fieldName != "id" && fieldName != "parent_id" {
		return nil, errors.Trace(hh.MakeInternalServerError(
			errors.Errorf("invalid fieldName: %q", fieldName),
		))
	}

	tagFields := "id, owner_id, parent_id, descr, children_cnt"
	var query string
	if !opts.GetNames {
		// No need to get tag names, so, just a simple query to the tags table
		query = fmt.Sprintf("SELECT %s FROM tags WHERE %s = $1", tagFields, fieldName)
	} else {
		// We need to get tag names, so here we add JSON array column with all
		// names (the first one is the primary one), and for ordering we also need
		// a separate JOIN which fetches just the primary name.
		//
		// Initially I tried to avoid the second JOIN, but ordering by a JSON
		// column works weird: e.g. if there are two rows:
		// - ["foo1", "foo2"]
		// - ["foo3"]
		// Then, for some reason, ["foo3"] becomes the first one.
		// If I try to order not by the whole column "names", but by names->0, then
		// it complains that there is no such column "names". So it seems it works
		// for real columns only, I don't know if it's intended.
		//
		// We could use ARRAY_AGG() instead of JSONB_AGG(), and ordering of array
		// works fine, but unmarshaling it is a pain.
		//
		// So, I resorted to the second JOIN and picking a primary name separately.
		// Plus, my measurements show that it even works faster than ordering by
		// the array column (by 10-15%)
		tagFields += ", JSONB_AGG((n.name) ORDER BY n.primary DESC) AS names"
		query = fmt.Sprintf(`
				SELECT %s FROM tags
				JOIN tag_names n ON n.tag_id = tags.id
				JOIN tag_names pn ON pn.tag_id = tags.id AND pn.primary = true
				WHERE %s = $1
				GROUP BY tags.id, pn.name
				ORDER BY pn.name`,
			tagFields, fieldName,
		)
	}

	rows, err := tx.Query(query, tagID)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, errors.Annotatef(
				hh.MakeInternalServerError(err),
				"getting tags with %s %d", fieldName, tagID,
			)
		}
		// No children
		return nil, nil
	}
	defer rows.Close()
	for rows.Next() {
		var td storage.TagData
		var childrenCnt int
		var pparentTagID *int
		var namesJSON []byte
		scan := []interface{}{
			&td.ID, &td.OwnerID, &pparentTagID, &td.Description, &childrenCnt,
		}
		if opts.GetNames {
			scan = append(scan, &namesJSON)
		}
		err := rows.Scan(scan...)
		if err != nil {
			return nil, errors.Trace(hh.MakeInternalServerError(err))
		}

		if opts.GetNames {
			json.Unmarshal(namesJSON, &td.Names)
		}

		if pparentTagID != nil {
			// There is a parent tag ID
			td.ParentTagID = pparentTagID
		} else {
			// There is no parent tag ID: use 0
			td.ParentTagID = cptr.Int(0)
		}

		tagsData = append(tagsData, td)
		childrenCntArr = append(childrenCntArr, childrenCnt)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Annotatef(err, "closing rows")
	}

	if opts.GetSubtags {
		for i, _ := range tagsData {
			if childrenCntArr[i] > 0 {
				td := &tagsData[i]
				td.Subtags, err = s.getTagsInternal(tx, "parent_id", td.ID, opts)
				if err != nil {
					return nil, errors.Trace(err)
				}
			}
		}
	}

	return tagsData, nil
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

type namesDiff struct {
	add          []string
	delete       []string
	setPrimary   *string
	clearPrimary *string
}

func (s *StoragePostgres) getNamesDiff(current, desired []string) *namesDiff {
	diff := namesDiff{}

	cm := make(map[string]struct{})
	dm := make(map[string]struct{})

	for _, k := range current {
		cm[k] = struct{}{}
	}

	for _, k := range desired {
		dm[k] = struct{}{}
	}

	for k := range dm {
		if _, ok := cm[k]; !ok {
			diff.add = append(diff.add, k)
		}
	}

	for k := range cm {
		if _, ok := dm[k]; !ok {
			diff.delete = append(diff.delete, k)
		}
	}

	if current[0] != desired[0] {
		// We'll need to set a new primary name
		diff.setPrimary = &desired[0]

		// We'll also need to clear a primary flag for the old primary name,
		// but if only this name is not going to be deleted at all
		if _, ok := dm[current[0]]; ok {
			diff.clearPrimary = &current[0]
		}
	}

	return &diff
}

func (s *StoragePostgres) addTagName(
	tx *sql.Tx, tagID, parentTagID int, name string, primary, allowEmpty bool,
) error {
	glog.V(3).Infof(
		"Adding tag name %q for tag %d, primary: %q", name, tagID, primary,
	)

	err := storage.ValidateTagName(name, allowEmpty)
	if err != nil {
		return errors.Trace(err)
	}

	// Check if tag with the given name already exists under the parent tag
	exists, err := s.tagExists(tx, parentTagID, name)
	if err != nil {
		return errors.Trace(err)
	}
	if exists {
		return errors.Errorf("Tag with the name %q already exists", name)
	}

	_, err = tx.Exec(
		`INSERT INTO tag_names (tag_id, name, "primary") VALUES ($1, $2, $3)`,
		tagID, name, primary,
	)
	if err != nil {
		return hh.MakeInternalServerError(errors.Annotatef(
			err, "adding tag name: %q for tag with id %d", name, tagID,
		))
	}

	return nil
}

func (s *StoragePostgres) deleteTagName(
	tx *sql.Tx, tagID int, name string,
) error {
	glog.V(3).Infof("Deleting tag name %q from tag %d", name, tagID)

	_, err := tx.Exec(
		`DELETE FROM tag_names WHERE tag_id = $1 and name = $2`,
		tagID, name,
	)
	if err != nil {
		return hh.MakeInternalServerError(errors.Annotatef(
			err, "deleting tag name: %q for tag with id %d", name, tagID,
		))
	}

	return nil
}

func (s *StoragePostgres) setTagNamePrimary(
	tx *sql.Tx, tagID int, name string, primary bool,
) error {
	glog.V(3).Infof(
		"Setting primariness of tag name %q from tag %d, primary: %q",
		name, tagID, primary,
	)

	_, err := tx.Exec(
		`UPDATE tag_names SET "primary" = $1 WHERE tag_id = $2 and name = $3`,
		primary, tagID, name,
	)
	if err != nil {
		return hh.MakeInternalServerError(errors.Annotatef(
			err, "updating tag name primariness: %q for tag with id %d, primary: %q",
			name, tagID, primary,
		))
	}

	return nil
}

// taghier's registry implementation which hits the database {{{
type thReg struct {
	s  *StoragePostgres
	tx *sql.Tx
}

func (r *thReg) GetParent(id int) (int, error) {
	td, err := r.s.GetTag(r.tx, id, &storage.GetTagOpts{
		GetNames:   false,
		GetSubtags: false,
	})
	if err != nil {
		return 0, errors.Trace(err)
	}

	return *td.ParentTagID, nil
}

// }}}
