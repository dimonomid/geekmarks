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

		// add bkm1 tagged with tag1/tag3
		bkm1ID, err := addBookmark(be, u1ID, &bkmData{
			URL:     "url_1",
			Title:   "title_1",
			Comment: "comment_1",
			TagIDs: []int{
				tagIDs.tag3ID,
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

		// get tagged with tag3: should return bkm1
		bkm1data, err := checkBkmGet(be, u1ID, []int{tagIDs.tag3ID}, []int{bkm1ID})
		if err != nil {
			return errors.Trace(err)
		}

		// check contents as well
		if bkm1data[0].URL != "url_1" {
			t.Errorf("bookmark url: expected %q, got %q", "url_1", bkm1data[0].URL)
		}

		if bkm1data[0].Title != "title_1" {
			t.Errorf("bookmark title: expected %q, got %q", "title_1", bkm1data[0].Title)
		}

		if bkm1data[0].Comment != "comment_1" {
			t.Errorf("bookmark comment: expected %q, got %q", "comment_1", bkm1data[0].Comment)
		}

		// get tagged with tag1: should return bkm1, bkm2
		_, err = checkBkmGet(be, u1ID, []int{tagIDs.tag1ID}, []int{bkm1ID, bkm2ID})
		if err != nil {
			return errors.Trace(err)
		}

		// get tagged with tag1, tag3: should return bkm1
		_, err = checkBkmGet(be, u1ID, []int{tagIDs.tag1ID, tagIDs.tag3ID}, []int{bkm1ID})
		if err != nil {
			return errors.Trace(err)
		}

		// get tagged with tag1, tag3, tag8: should return nothing
		_, err = checkBkmGet(be, u1ID, []int{tagIDs.tag1ID, tagIDs.tag3ID, tagIDs.tag8ID}, []int{})
		if err != nil {
			return errors.Trace(err)
		}

		fmt.Println(tagIDs.tag1ID, bkm1ID, bkm2ID)

		return nil
	})
}

type bkmData struct {
	ID        int    `json:"id"`
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	Comment   string `json:"comment,omitempty"`
	UpdatedAt uint64 `json:"updatedAt"`
	TagIDs    []int  `json:"tagIDs"`
}

type bkms []bkmData

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

func checkBkmGet(be testBackend, userID int, tagIDs []int, expectedBkmIDs []int) ([]bkmData, error) {

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

	return []bkmData(v), nil
}
