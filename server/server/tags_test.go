// +build all_tests integration_tests

package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"
	"reflect"
	"testing"

	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/testutils"
	"github.com/juju/errors"
)

func TestTagsByPattern(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		var u1ID int
		var err error

		if u1ID, err = testutils.CreateTestUser(t, si, "test1", "1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", "1")

		_, err = makeTestTagsHierarchy(be, u1ID)
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "g7", false, []string{
			"/tag7",
			"/tag7/tag8",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "g7", true, []string{
			"/tag7",
			"/g7 NEWTAGS(1)",
			"/tag7/tag8",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7", true, []string{
			"/tag7",
			"/tag7/tag8",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7/g8", true, []string{
			"/tag7/tag8",
			"/tag7/g8 NEWTAGS(1)",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7/g8/g88", true, []string{
			"/tag7/g8/g88 NEWTAGS(2)",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7////g8/g88", true, []string{
			"/tag7/g8/g88 NEWTAGS(2)",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7  /    g8     ", true, []string{
			"/tag7/tag8",
			"/tag7/g8 NEWTAGS(1)",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7/g= 8", true, []string{
			"/tag7/g-8 NEWTAGS(1)",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7/===g===8===", true, []string{
			"/tag7/g-8 NEWTAGS(1)",
		})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7/---g---8---", true, []string{
			"/tag7/g-8 NEWTAGS(1)",
		})
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
}

type tagData struct {
	Path        string `json:"path"`
	ID          int    `json:"id"`
	Description string `json:"description"`
	NewTagsCnt  int    `json:"newTagsCnt"`
}

func checkTagsGet(
	be testBackend, userID int, pattern string, allowNew bool, expectedPaths []string,
) ([]tagData, error) {

	qsVals := url.Values{}
	qsVals.Add("pattern", pattern)

	if allowNew {
		qsVals.Add("allow_new", "1")
	}

	resp, err := be.DoUserReq(
		"GET", "/tags?"+qsVals.Encode(), userID, nil, true,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Trace(err)
	}

	v := []tagData{}
	err = json.Unmarshal(body, &v)
	if err != nil {
		fmt.Printf("body: %q\n", body)
		return nil, errors.Trace(err)
	}

	gotPaths := []string{}
	for _, b := range v {
		p := b.Path
		if b.NewTagsCnt > 0 {
			p += fmt.Sprintf(" NEWTAGS(%d)", b.NewTagsCnt)
		}
		gotPaths = append(gotPaths, p)
	}

	if !reflect.DeepEqual(gotPaths, expectedPaths) {
		return nil, errors.Errorf("tags mismatch: expectedPaths %v, got %v",
			expectedPaths, gotPaths,
		)
	}

	return v, nil
}
