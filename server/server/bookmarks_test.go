// +build all_tests integration_tests

package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	"strings"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/testutils"
	"github.com/juju/errors"
)

func TestBookmarks(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		var u1ID int
		var err error

		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", "1")

		tagIDs, err := makeTestTagsHierarchy(be, u1ID)
		if err != nil {
			return errors.Trace(err)
		}

		// add bkm1 tagged with tag1/tag3 and tag8
		bkm1ID, err := addBookmark(be, u1ID, &bkmData{
			URL:     "url_1",
			Title:   "title_1",
			Comment: "comment_1",
			TagIDs: []int{
				tagIDs.tag3ID,
				tagIDs.tag8ID,
			},
		})
		if err != nil {
			return errors.Trace(err)
		}

		// add bkm2 tagged with tag1
		bkm2ID, err := addBookmark(be, u1ID, &bkmData{
			URL:     "url_2",
			Title:   "title_2",
			Comment: "comment_2",
			TagIDs: []int{
				tagIDs.tag1ID,
			},
		})
		if err != nil {
			return errors.Trace(err)
		}

		err = checkBkmGetByID(be, u1ID, bkm1ID, &bkmData{
			ID:      bkm1ID,
			URL:     "url_1",
			Title:   "title_1",
			Comment: "comment_1",
			Tags: []bkmTagData{
				bkmTagData{ID: tagIDs.tag3ID, FullName: "/tag1/tag3_alias"},
				bkmTagData{ID: tagIDs.tag8ID, FullName: "/tag7/tag8"},
			},
		})
		if err != nil {
			return errors.Trace(err)
		}

		// get tagged with tag3: should return bkm1
		bkmRespData, err := checkBkmGet(be, u1ID, []int{tagIDs.tag3ID}, []int{bkm1ID})
		if err != nil {
			return errors.Trace(err)
		}

		// check contents as well
		if got, want := bkmRespData[0].URL, "url_1"; got != want {
			t.Errorf("bookmark url: got %q, want %q", got, want)
		}

		if got, want := bkmRespData[0].Title, "title_1"; got != want {
			t.Errorf("bookmark title: got %q, want %q", got, want)
		}

		if got, want := bkmRespData[0].Comment, "comment_1"; got != want {
			t.Errorf("bookmark comment: got %q, want %q", got, want)
		}

		if err := checkBkmTags(&bkmRespData[0], []bkmTagData{
			bkmTagData{ID: tagIDs.tag3ID, FullName: "/tag1/tag3_alias"},
			bkmTagData{ID: tagIDs.tag8ID, FullName: "/tag7/tag8"},
		}); err != nil {
			return errors.Trace(err)
		}

		// get tagged with tag1: should return bkm1, bkm2
		bkmRespData, err = checkBkmGet(be, u1ID, []int{tagIDs.tag1ID}, []int{bkm1ID, bkm2ID})
		if err != nil {
			return errors.Trace(err)
		}

		if err := checkBkmTags(&bkmRespData[0], []bkmTagData{
			bkmTagData{ID: tagIDs.tag3ID, FullName: "/tag1/tag3_alias"},
			bkmTagData{ID: tagIDs.tag8ID, FullName: "/tag7/tag8"},
		}); err != nil {
			return errors.Trace(err)
		}

		if err := checkBkmTags(&bkmRespData[1], []bkmTagData{
			bkmTagData{ID: tagIDs.tag1ID, FullName: "/tag1"},
		}); err != nil {
			return errors.Trace(err)
		}

		// get tagged with tag1, tag3: should return bkm1
		_, err = checkBkmGet(be, u1ID, []int{tagIDs.tag1ID, tagIDs.tag3ID}, []int{bkm1ID})
		if err != nil {
			return errors.Trace(err)
		}

		// get tagged with tag1, tag3, tag2: should return nothing
		_, err = checkBkmGet(be, u1ID, []int{tagIDs.tag1ID, tagIDs.tag3ID, tagIDs.tag2ID}, []int{})
		if err != nil {
			return errors.Trace(err)
		}

		fmt.Println(tagIDs.tag1ID, bkm1ID, bkm2ID)

		return nil
	})
}

type bkmData struct {
	ID        int          `json:"id"`
	URL       string       `json:"url"`
	Title     string       `json:"title,omitempty"`
	Comment   string       `json:"comment,omitempty"`
	UpdatedAt uint64       `json:"updatedAt"`
	TagIDs    []int        `json:"tagIDs"`
	Tags      []bkmTagData `json:"tags,omitempty"`
}

type bkmTagsByID []bkmTagData

type bkmTagData struct {
	ID       int    `json:"id"`
	ParentID int    `json:"parentID,omitempty"`
	Name     string `json:"name,omitempty"`
	FullName string `json:"fullName,omitempty"`
}

func (s bkmTagsByID) Len() int {
	return len(s)
}
func (s bkmTagsByID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s bkmTagsByID) Less(i, j int) bool {
	return s[i].ID < s[j].ID
}

type bkms []bkmData
type bkmsByID bkms

func (s bkmsByID) Len() int {
	return len(s)
}
func (s bkmsByID) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s bkmsByID) Less(i, j int) bool {
	return s[i].ID < s[j].ID
}

func addBookmark(be testBackend, userID int, data *bkmData) (bkmID int, err error) {
	tagIDs := A{}
	for _, id := range data.TagIDs {
		tagIDs = append(tagIDs, id)
	}
	resp, err := be.DoUserReq("POST", "/bookmarks", userID, H{
		"url":     data.URL,
		"title":   data.Title,
		"comment": data.Comment,
		"tagIDs":  tagIDs,
	}, true)
	if err != nil {
		return 0, errors.Trace(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, errors.Trace(err)
	}

	v := map[string]int{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return v["bookmarkID"], nil
}

func checkBkmGet(
	be testBackend, userID int, tagIDs []int, expectedBkmIDs []int,
) ([]bkmData, error) {

	qsParts := []string{}
	for _, tagID := range tagIDs {
		qsParts = append(qsParts, fmt.Sprintf("tag_id=%d", tagID))
	}

	resp, err := be.DoUserReq(
		"GET", "/bookmarks?"+strings.Join(qsParts, "&"), userID, nil, true,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	v := bkms{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		fmt.Printf("body: %q\n", body)
		return nil, errors.Trace(err)
	}

	bkmIDs := []int{}
	for _, b := range v {
		bkmIDs = append(bkmIDs, b.ID)
	}

	sort.Ints(bkmIDs)
	sort.Ints(expectedBkmIDs)

	if !reflect.DeepEqual(bkmIDs, expectedBkmIDs) {
		return nil, errors.Errorf("bookmarks mismatch: expected %v, got %v", expectedBkmIDs, bkmIDs)
	}

	sort.Sort(bkmsByID(v))

	return []bkmData(v), nil
}

func checkBkmGetByID(be testBackend, userID int, bkmID int, expectedBkm *bkmData) error {
	resp, err := be.DoUserReq(
		"GET", fmt.Sprintf("/bookmarks/%d", bkmID), userID, nil, true,
	)
	if err != nil {
		return errors.Trace(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Trace(err)
	}

	v := bkmData{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		fmt.Printf("body: %q\n", body)
		return errors.Trace(err)
	}

	// don't compare UpdatedAt
	v.UpdatedAt = 0

	sort.Sort(bkmTagsByID(v.Tags))
	sort.Sort(bkmTagsByID(expectedBkm.Tags))

	if !reflect.DeepEqual(&v, expectedBkm) {
		return errors.Errorf("bookmark mismatches: expected %v, got %v", expectedBkm, v)
	}

	return nil
}

func checkBkmTags(bkm *bkmData, expectedTags []bkmTagData) error {
	sort.Sort(bkmTagsByID(expectedTags))
	sort.Sort(bkmTagsByID(bkm.Tags))

	if !reflect.DeepEqual(expectedTags, bkm.Tags) {
		return errors.Errorf("bookmark tags mismatch: expected %v, got %v", expectedTags, bkm.Tags)
	}

	return nil
}
