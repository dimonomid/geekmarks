package postgres

import (
	"database/sql"
	"strings"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"
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

func (s *StoragePostgres) UpdateTag(tx *sql.Tx, td *storage.TagData) (err error) {
	if td.ParentTagID != nil {
		// TODO
		return errors.Errorf("moving tags is not yet implemented")
	}

	// If description is provided, update it
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

	// Let's see if we need to update names
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

	return nil
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
				interror.WrapInternalError(
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
	if fieldName != "id" && fieldName != "parent_id" {
		return nil, errors.Trace(hh.MakeInternalServerError(
			errors.Errorf("invalid fieldName: %q", fieldName),
		))
	}
	rows, err := tx.Query(
		"SELECT id, owner_id, parent_id, descr FROM tags WHERE "+fieldName+" = $1",
		tagID,
	)
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
		var pparentTagID *int
		err := rows.Scan(&td.ID, &td.OwnerID, &pparentTagID, &td.Description)
		if err != nil {
			return nil, errors.Trace(hh.MakeInternalServerError(err))
		}

		if pparentTagID != nil {
			// There is a parent tag ID
			td.ParentTagID = pparentTagID
		} else {
			// There is no parent tag ID: use 0
			td.ParentTagID = cptr.Int(0)
		}

		tagsData = append(tagsData, td)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Annotatef(err, "closing rows")
	}

	if opts.GetNames {
		for i, _ := range tagsData {
			td := &tagsData[i]
			td.Names, err = s.GetTagNames(tx, td.ID)
			if err != nil {
				return nil, errors.Trace(err)
			}
		}
	}

	if opts.GetSubtags {
		for i, _ := range tagsData {
			td := &tagsData[i]
			td.Subtags, err = s.getTagsInternal(tx, "parent_id", td.ID, opts)
			if err != nil {
				return nil, errors.Trace(err)
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
