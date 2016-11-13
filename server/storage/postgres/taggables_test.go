// +build all_tests integration_tests

package postgres

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/testutils"

	"github.com/juju/errors"
)

func TestTaggables(t *testing.T) {
	runWithRealDB(t, func(si *StoragePostgres) error {
		var u1ID, u2ID int
		var err error
		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		if u2ID, err = testutils.CreateTestUser(t, si, "test2", "2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}

		err = si.Tx(func(tx *sql.Tx) error {
			u1TagIDs, err := makeTagsHierarchy(tx, si, u1ID)
			if err != nil {
				return errors.Annotatef(err, "creating test tags hierarchy for user1")
			}

			u2TagIDs, err := makeTagsHierarchy(tx, si, u2ID)
			if err != nil {
				return errors.Annotatef(err, "creating test tags hierarchy for user2")
			}

			bkm1ID, err := si.CreateBookmark(tx, &storage.BookmarkData{
				OwnerID: u1ID,
				URL:     "url1",
				Title:   "title1",
				Comment: "comment1",
			})
			if err != nil {
				return errors.Annotatef(err, "creating bookmark")
			}

			bkm2ID, err := si.CreateBookmark(tx, &storage.BookmarkData{
				OwnerID: u1ID,
				URL:     "url2",
				Title:   "title2",
				Comment: "comment2",
			})
			if err != nil {
				return errors.Annotatef(err, "creating bookmark")
			}

			// tag bkm1 with tag1/tag3
			err = si.SetTaggings(
				tx, bkm1ID, []int{u1TagIDs.tag3ID}, storage.TaggingModeLeafs,
			)
			if err != nil {
				return errors.Trace(err)
			}

			// tag bkm2 with tag1
			err = si.SetTaggings(
				tx, bkm2ID, []int{u1TagIDs.tag1ID}, storage.TaggingModeLeafs,
			)
			if err != nil {
				return errors.Trace(err)
			}

			// Tagged with tag3: should return bkm1
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag3ID}, nil, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{bkm1ID}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			// Tagged bookmarks with tag3: should return bkm1
			{
				bkms, err := si.GetTaggedBookmarks(
					tx, []int{u1TagIDs.tag3ID}, nil, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if len(bkms) != 1 {
					return errors.Errorf("should get 1 bookmark")
				}

				if bkms[0].URL != "url1" {
					return errors.Errorf("URL: expected url1, got %q", bkms[0].URL)
				}

				if bkms[0].Title != "title1" {
					return errors.Errorf("Title: expected title1, got %q", bkms[0].Title)
				}

				if bkms[0].Comment != "comment1" {
					return errors.Errorf("Comment: expected comment1, got %q", bkms[0].Comment)
				}
			}

			// Tagged with tag1: should return bkm1, bkm2
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag1ID}, nil, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{bkm1ID, bkm2ID}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			// Tagged with tag1, tag3: should return bkm1
			// (also we specify taggable type: bookmark; which shouldn't make any difference)
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag1ID, u1TagIDs.tag3ID}, nil, []storage.TaggableType{storage.TaggableTypeBookmark},
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{bkm1ID}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			// Tagged with tag1, tag3, tag8: should return nothing
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag1ID, u1TagIDs.tag3ID, u1TagIDs.tag8ID}, nil, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			// tag bkm1 with tag1/tag3, tag7/tag8 (i.e. add tag7/tag8)
			err = si.SetTaggings(
				tx, bkm1ID, []int{u1TagIDs.tag3ID, u1TagIDs.tag8ID}, storage.TaggingModeLeafs,
			)
			if err != nil {
				return errors.Trace(err)
			}

			// Tagged with tag1, tag3: should return bkm1
			// (also we specify user id, which should not make any difference)
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag1ID, u1TagIDs.tag3ID}, &u1ID, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{bkm1ID}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			// Tagged with tag1, tag3, tag8: should return bkm1
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag1ID, u1TagIDs.tag3ID, u1TagIDs.tag8ID}, nil, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{bkm1ID}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			// tag bkm1 with tag1, tag7/tag8 (i.e. remove tag3)
			err = si.SetTaggings(
				tx, bkm1ID, []int{u1TagIDs.tag1ID, u1TagIDs.tag8ID}, storage.TaggingModeLeafs,
			)
			if err != nil {
				return errors.Trace(err)
			}

			// Tagged with tag1, tag3, tag8: should return nothing
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag1ID, u1TagIDs.tag3ID, u1TagIDs.tag8ID}, nil, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			// Tagged with tag1, tag8: should return bkm1
			{
				taggableIDs, err := si.GetTaggedTaggableIDs(
					tx, []int{u1TagIDs.tag1ID, u1TagIDs.tag8ID}, nil, nil,
				)
				if err != nil {
					return errors.Trace(err)
				}
				if err := checkTgb(taggableIDs, []int{bkm1ID}); err != nil {
					t.Errorf("%s", errors.Trace(err))
				}
			}

			fmt.Println(u1TagIDs, u2TagIDs, bkm1ID)

			return nil
		})
		return errors.Trace(err)
	})
}

func checkTgb(got, expected []int) error {
	if expected == nil {
		expected = []int{}
	}

	if got == nil {
		got = []int{}
	}

	sort.Ints(expected)
	sort.Ints(got)

	if !reflect.DeepEqual(got, expected) {
		return errors.Errorf("taggables mismatch: expected %v, got %v", expected, got)
	}

	return nil
}
