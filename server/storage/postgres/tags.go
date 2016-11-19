package postgres

import (
	"database/sql"
	"strings"

	"dmitryfrank.com/geekmarks/server/cptr"
	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/interror"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) CreateTag(
	tx *sql.Tx, td *storage.TagData,
) (tagID int, err error) {
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

	for i, name := range td.Names {
		primary := false
		if i == 0 {
			primary = true
		}
		err := storage.ValidateTagName(name, pParent == nil)
		if err != nil {
			return 0, errors.Trace(err)
		}

		// Check if tag with the given name already exists under the parent tag
		exists, err := s.tagExists(tx, td.ParentTagID, name)
		if err != nil {
			return 0, errors.Trace(err)
		}
		if exists {
			return 0, errors.Errorf("Tag with the name %q already exists", name)
		}

		_, err = tx.Exec(
			`INSERT INTO tag_names (tag_id, name, "primary") VALUES ($1, $2, $3)`,
			tagID, name, primary,
		)
		if err != nil {
			return 0, hh.MakeInternalServerError(errors.Annotatef(
				err, "adding tag name: %q for tag with id %d", name, tagID,
			))
		}
	}

	for _, subTag := range td.Subtags {
		_, err := s.CreateTag(tx, &subTag)
		if err != nil {
			return 0, errors.Annotatef(err, "creating subtag")
		}
	}

	return tagID, nil
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
			td.ParentTagID = *pparentTagID
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
