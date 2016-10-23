package postgres

import (
	"database/sql"
	"fmt"
	"strconv"

	hh "dmitryfrank.com/geekmarks/server/httphelper"
	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

func (s *StoragePostgres) CreateTaggable(tx *sql.Tx, tgbd *storage.TaggableData) (tgbID int, err error) {
	err = tx.QueryRow(
		"INSERT INTO taggables (owner_id, type) VALUES ($1, $2) RETURNING id",
		tgbd.OwnerID, string(tgbd.Type),
	).Scan(&tgbID)
	if err != nil {
		return 0, hh.MakeInternalServerError(errors.Annotatef(
			err, "adding new taggable (owner_id: %d, type: %s)", tgbd.OwnerID, tgbd.Type,
		))
	}

	return tgbID, nil
}

func (s *StoragePostgres) GetTaggedTaggableIDs(
	tx *sql.Tx, tagIDs []int, ownerID *int, ttypes []storage.TaggableType,
) (taggableIDs []int, err error) {
	args := []interface{}{}
	phNum := 1

	// Build query
	query := "SELECT id FROM taggables "

	for k, tagID := range tagIDs {
		query += fmt.Sprintf(
			"JOIN taggings t%d ON (t%d.taggable_id = taggables.id AND t%d.tag_id = $%d) ",
			k, k, k,
			phNum,
		)
		phNum++
		args = append(args, tagID)
	}

	query += "WHERE 1=1 "

	if ownerID != nil {
		query += fmt.Sprintf("AND owner_id = $%d ", phNum)
		phNum++
		args = append(args, *ownerID)
	}

	if len(ttypes) > 0 {
		qtmp := ""
		for i, ttype := range ttypes {
			if i > 0 {
				qtmp += "OR "
			}
			qtmp += fmt.Sprintf("type = $%d ", phNum)
			phNum++
			args = append(args, string(ttype))
		}
		query += "AND ( " + qtmp + " ) "
	}

	// Execute it
	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, hh.MakeInternalServerError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var taggableID int
		err := rows.Scan(&taggableID)
		if err != nil {
			return nil, hh.MakeInternalServerError(err)
		}
		taggableIDs = append(taggableIDs, taggableID)
	}
	if err := rows.Close(); err != nil {
		return nil, errors.Annotatef(err, "closing rows")
	}

	return taggableIDs, nil
}

func getPlaceholdersString(start, cnt int) string {
	ret := ""

	for i := start; i < (start + cnt); i++ {
		if ret != "" {
			ret += ","
		}
		ret += "$" + strconv.Itoa(i)
	}

	return ret
}
