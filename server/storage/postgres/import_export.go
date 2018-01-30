// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package postgres

import (
	"database/sql"
	"fmt"

	"github.com/juju/errors"

	"dmitryfrank.com/geekmarks/server/storage"
)

type pgTagDump struct {
	ParentTagID int      `json:"parent"`
	Description string   `json:"description,omitempty"`
	Names       []string `json:"names"`
}

type pgBookmarkDump struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Comment string `json:"comment,omitempty"`
	// TODO: createdat, updatedat
	Tags []int `json:"tags"`
}

type pgDataDump struct {
	Tags      map[int]pgTagDump      `json:"tags"`
	Bookmarks map[int]pgBookmarkDump `json:"bookmark"`
}

func (s *StoragePostgres) Export(tx *sql.Tx, ownerID int) (dump *storage.DataDump, err error) {
	dump = &storage.DataDump{
		StorageType: storage.StorageTypePostgres,
		DumpVersion: "2017-06-18",
	}

	pgDump := pgDataDump{
		Tags:      make(map[int]pgTagDump),
		Bookmarks: make(map[int]pgBookmarkDump),
	}

	tagData, err := s.getTagsInternal(tx, "owner_id", ownerID, &storage.GetTagOpts{
		GetNames: true,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	allBkm, err := s.getAllBookmarks(tx, ownerID, &storage.TagsFetchOpts{
		TagsFetchMode: storage.TagsFetchModeLeafs,
		// TODO: fix getAllBookmarks() for TagNamesFetchModeNone, and uncomment
		// the line below: we don't really need names here
		//TagNamesFetchMode: storage.TagNamesFetchModeNone,
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	for _, t := range tagData {
		pgDump.Tags[t.ID] = pgTagDump{
			ParentTagID: *t.ParentTagID,
			Description: *t.Description,
			Names:       t.Names,
		}
	}

	for _, b := range allBkm {
		fmt.Println(b.Tags)

		tagsMap := map[int]struct{}{}
		tags := []int{}
		for _, t := range b.Tags {
			for _, ti := range t.TagItems {
				if _, ok := tagsMap[ti.ID]; !ok {
					tagsMap[ti.ID] = struct{}{}
					tags = append(tags, ti.ID)
				}
			}
		}
		pgDump.Bookmarks[b.ID] = pgBookmarkDump{
			URL:     b.URL,
			Title:   b.Title,
			Comment: b.Comment,
			Tags:    tags,
		}
	}

	dump.Data = pgDump
	return dump, nil
}
