// +build all_tests integration_tests

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"dmitryfrank.com/geekmarks/server/cptr"
	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/testutils"
	"github.com/juju/errors"
)

func TestTagsGet(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		ts := be.GetTestServer()
		var u1ID, u2ID int
		var u1Token, u2Token string
		var err error

		if u1ID, u1Token, err = testutils.CreateTestUser(t, si, "test1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", u1Token)

		if u2ID, u2Token, err = testutils.CreateTestUser(t, si, "test2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u2ID, "test2", u2Token)

		var u1TagsGetRespByPath, u1TagsGetRespByMy []byte
		var u2TagsGetRespByPath, u2TagsGetRespByMy []byte

		// test1 requests its own tags
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/users/%d/tags", ts.URL, u1ID), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.Header.Set("Authorization", "Bearer "+u1Token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u1TagsGetRespByPath, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// test1 requests its own tags via /api/my
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/my/tags", ts.URL), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.Header.Set("Authorization", "Bearer "+u1Token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u1TagsGetRespByMy, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// test1 requests FOREIGN tags, should fail
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/users/%d/tags", ts.URL, u2ID), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.Header.Set("Authorization", "Bearer "+u1Token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			genResp, err := makeGenericRespFromHTTPResp(resp)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(genResp, http.StatusForbidden, "forbidden"); err != nil {
				return errors.Trace(err)
			}
		}

		// test2 requests its own tags
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/users/%d/tags", ts.URL, u2ID), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			fmt.Println("set auth:", "Bearer "+u2Token)
			req.Header.Set("Authorization", "Bearer "+u2Token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u2TagsGetRespByPath, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// test2 requests its own tags via /api/my
		{
			req, err := http.NewRequest(
				"GET", fmt.Sprintf("%s/api/my/tags", ts.URL), nil,
			)
			if err != nil {
				return errors.Trace(err)
			}
			req.Header.Set("Authorization", "Bearer "+u2Token)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectHTTPCode2(resp, http.StatusOK); err != nil {
				return errors.Trace(err)
			}

			u2TagsGetRespByMy, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// check that responses match and mismatch as expected

		if string(u1TagsGetRespByPath) != string(u1TagsGetRespByMy) {
			return errors.Errorf("u1TagsGetRespByPath should be equal to u1TagsGetRespByMy")
		}

		if string(u2TagsGetRespByPath) != string(u2TagsGetRespByMy) {
			return errors.Errorf("u2TagsGetRespByPath should be equal to u2TagsGetRespByMy")
		}

		if string(u1TagsGetRespByPath) == string(u2TagsGetRespByPath) {
			return errors.Errorf("u1TagsGetRespByPath should NOT be equal to u2TagsGetRespByPath")
		}

		return nil
	})
}

// Ignores IDs
func tagDataEqual(tdExpected, tdGot *userTagData) error {
	if tdExpected.Description != tdGot.Description {
		return errors.Errorf("expected tag descr %q, got %q", tdExpected.Description, tdGot.Description)
	}

	if !reflect.DeepEqual(tdExpected.Names, tdGot.Names) {
		return errors.Errorf("expected names %v, got %v", tdExpected.Names, tdGot.Names)
	}

	if len(tdExpected.Subtags) != len(tdGot.Subtags) {
		return errors.Errorf(
			"expected subtags len %d, got %d (expected: %q, got: %q)",
			len(tdExpected.Subtags), len(tdGot.Subtags),
			tdExpected.Subtags, tdGot.Subtags,
		)
	}

	for k, _ := range tdExpected.Subtags {
		if err := tagDataEqual(&tdExpected.Subtags[k], &tdGot.Subtags[k]); err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}

func addTag(
	be testBackend, url string, userID int, names []string, descr string, createIntermediary bool,
) (int, error) {
	resp, err := be.DoUserReq(
		"POST", url, userID,
		H{
			"names":              names,
			"description":        descr,
			"createIntermediary": createIntermediary,
		},
		true,
	)
	if err != nil {
		return 0, errors.Trace(err)
	}

	var respMap map[string]interface{}
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&respMap)

	tagID, ok := respMap["tagID"]
	if !ok {
		return 0, errors.Errorf("response %v does not contain tagID", respMap)
	}
	if tagID.(float64) <= 0 {
		return 0, errors.Errorf("tagID should be > 0, but got %d", tagID)
	}
	return int(tagID.(float64)), nil
}

func updateTag(
	be testBackend, url string, userID int, names []string, descr *string,
	parentTagID *int,
) error {
	_, err := be.DoUserReq(
		"PUT", url, userID,
		H{
			"names":       names,
			"description": descr,
			"parentTagID": parentTagID,
		},
		true,
	)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func deleteTag(
	be testBackend, url string, userID int,
) error {
	_, err := be.DoUserReq(
		"DELETE", url, userID, nil, true,
	)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

type tagIDs struct {
	rootTagID, tag1ID, tag2ID, tag3ID, tag4ID, tag5ID, tag6ID, tag7ID, tag8ID int
}

// makeTestTagsHierarchy creates the following tag hierarchy for the given user:
// /
// ├── tag1
// │   └── tag3
// │       ├── tag4
// │       └── tag5
// │           └── tag6
// ├── tag2
// └── tag7
//     └── tag8
func makeTestTagsHierarchy(be testBackend, userID int) (ids *tagIDs, err error) {
	ids = &tagIDs{}
	ids.tag1ID, err = addTag(
		be, "/tags", userID, []string{"tag1", "tag1_alias"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.tag2ID, err = addTag(
		be, "/tags", userID, []string{"tag2", "tag2_alias"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.tag3ID, err = addTag(
		be, "/tags/tag1", userID, []string{"tag3_alias", "tag3"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.tag4ID, err = addTag(
		be, "/tags/tag1/tag3", userID, []string{"tag4", "tag4_alias"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.tag5ID, err = addTag(
		be, "/tags/tag1/tag3", userID, []string{"tag5", "tag5_alias"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.tag6ID, err = addTag(
		be, "/tags/tag1/tag3/tag5", userID, []string{"tag6", "tag6_alias"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.tag7ID, err = addTag(
		be, "/tags", userID, []string{"tag7", "tag7_alias"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.tag8ID, err = addTag(
		be, "/tags/tag7", userID, []string{"tag8", "tag8_alias"}, "test tag", false,
	)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return ids, nil
}

type bkmIDs struct {
	bkm1ID, bkm2ID, bkm3ID, bkm4ID, bkm5ID, bkm6ID, bkm7ID, bkm8ID, bkm2_5ID, bkm4_5ID, bkm_untagged_ID int
}

func makeTestBookmarks(be testBackend, userID int, tagIDs *tagIDs) (ids *bkmIDs, err error) {
	ids = &bkmIDs{}

	ids.bkm1ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_1",
		Title:   "title_tag_1",
		Comment: "comment_tag_1",
		TagIDs: []int{
			tagIDs.tag1ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm2ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_2",
		Title:   "title_tag_2",
		Comment: "comment_tag_2",
		TagIDs: []int{
			tagIDs.tag2ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm3ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_3",
		Title:   "title_tag_3",
		Comment: "comment_tag_3",
		TagIDs: []int{
			tagIDs.tag3ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm4ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_4",
		Title:   "title_tag_4",
		Comment: "comment_tag_4",
		TagIDs: []int{
			tagIDs.tag4ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm5ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_5",
		Title:   "title_tag_5",
		Comment: "comment_tag_5",
		TagIDs: []int{
			tagIDs.tag5ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm6ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_6",
		Title:   "title_tag_6",
		Comment: "comment_tag_6",
		TagIDs: []int{
			tagIDs.tag6ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm7ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_7",
		Title:   "title_tag_7",
		Comment: "comment_tag_7",
		TagIDs: []int{
			tagIDs.tag7ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm8ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_8",
		Title:   "title_tag_8",
		Comment: "comment_tag_8",
		TagIDs: []int{
			tagIDs.tag8ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm2_5ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_2_5",
		Title:   "title_tag_2_5",
		Comment: "comment_tag_2_5",
		TagIDs: []int{
			tagIDs.tag2ID,
			tagIDs.tag5ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm4_5ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_tag_4_5",
		Title:   "title_tag_4_5",
		Comment: "comment_tag_4_5",
		TagIDs: []int{
			tagIDs.tag4ID,
			tagIDs.tag5ID,
		},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	ids.bkm_untagged_ID, err = addBookmark(be, userID, &bkmData{
		URL:     "url_untagged",
		Title:   "title_untagged",
		Comment: "comment_untagged",
		TagIDs:  []int{},
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return ids, nil
}

func TestTagsGetSet(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		var u1ID, u2ID int
		var u1Token, u2Token string
		var err error

		if u1ID, u1Token, err = testutils.CreateTestUser(t, si, "test1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", u1Token)

		if u2ID, u2Token, err = testutils.CreateTestUser(t, si, "test2", "2@2.2"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u2ID, "test2", u2Token)

		var tagID_Foo1, tagID_Foo3, tagID_Foo1_a, tagID_Foo1_b, tagID_Foo1_b_c int

		// Get initial tag tree (should be only root tag)
		{
			resp, err := be.DoUserReq("GET", "/tags", u1ID, nil, true)
			if err != nil {
				return errors.Trace(err)
			}

			var tdGot userTagData
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&tdGot)

			tdExpected := userTagData{
				Names:       []string{""},
				Description: "Root pseudo-tag",
				Subtags:     []userTagData{},
			}

			err = tagDataEqual(&tdExpected, &tdGot)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// Try to add tag foo1 (foo2)
		tagID_Foo1, err = addTag(
			be, "/tags", u1ID, []string{"foo1", "foo2"}, "my foo descr", false,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag which already exists (should fail)
		{
			resp, err := be.DoUserReq(
				"POST", "/tags", u1ID,
				H{"names": A{"foo3", "foo2", "foo4"}},
				false,
			)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(
				resp, http.StatusBadRequest, "Tag with the name \"foo2\" already exists",
			); err != nil {
				return errors.Trace(err)
			}
		}

		// Try to add tag for another user (should fail)
		{
			resp, err := be.DoReq(
				"POST", fmt.Sprintf("/api/users/%d/tags", u2ID), u1Token,
				bytes.NewReader([]byte(`
				{"names": ["test"]}
				`)),
				false,
			)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(
				resp, http.StatusForbidden, "forbidden",
			); err != nil {
				return errors.Trace(err)
			}
		}

		// Try to add tag foo3
		tagID_Foo3, err = addTag(
			be, "/tags", u1ID, []string{"foo3"}, "my foo 3 tag", false,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo1 / a
		tagID_Foo1_a, err = addTag(
			be, "/tags/foo1", u1ID, []string{"a"}, "", false,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo2 / b (note that foo1 is the same as foo2)
		tagID_Foo1_b, err = addTag(
			be, "/tags/foo2", u1ID, []string{"b"}, "", false,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo2 / b / Привет, specifying parent as ID, not path
		tagID_Foo1_b_c, err = addTag(
			be, fmt.Sprintf("/tags/%d", tagID_Foo1_b), u1ID, []string{"Привет"}, "", false,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag foo1 / bar1 / bar2 / bar3 (three new tags at once)
		_, err = addTag(
			be, "/tags/foo1/bar1/bar2", u1ID, []string{"bar3"}, "", true,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add tag multiple tags at once starting from the root:
		// hey1 / hey2 / hey3
		_, err = addTag(
			be, "/tags/hey1/hey2", u1ID, []string{"hey3"}, "", true,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Try to add single tag starting from the root: hey_root
		_, err = addTag(
			be, "/tags", u1ID, []string{"hey_root"}, "", true,
		)
		if err != nil {
			return errors.Trace(err)
		}

		// Get resulting tag tree
		{
			resp, err := be.DoUserReq(
				"GET", "/tags", u1ID, nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			var tdGot userTagData
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&tdGot)

			tdExpected := userTagData{
				Names:       []string{""},
				Description: "Root pseudo-tag",
				Subtags: []userTagData{
					userTagData{
						Names:       []string{"foo1", "foo2"},
						Description: "my foo descr",
						Subtags: []userTagData{
							userTagData{
								Names:       []string{"a"},
								Description: "",
								Subtags:     []userTagData{},
							},
							userTagData{
								Names:       []string{"b"},
								Description: "",
								Subtags: []userTagData{
									userTagData{
										Names:       []string{"Привет"},
										Description: "",
										Subtags:     []userTagData{},
									},
								},
							},
							userTagData{
								Names:       []string{"bar1"},
								Description: "",
								Subtags: []userTagData{
									userTagData{
										Names:       []string{"bar2"},
										Description: "",
										Subtags: []userTagData{
											userTagData{
												Names:       []string{"bar3"},
												Description: "",
												Subtags:     []userTagData{},
											},
										},
									},
								},
							},
						},
					},
					userTagData{
						Names:       []string{"foo3"},
						Description: "my foo 3 tag",
						Subtags:     []userTagData{},
					},
					userTagData{
						Names:       []string{"hey1"},
						Description: "",
						Subtags: []userTagData{
							userTagData{
								Names:       []string{"hey2"},
								Description: "",
								Subtags: []userTagData{
									userTagData{
										Names:       []string{"hey3"},
										Description: "",
										Subtags:     []userTagData{},
									},
								},
							},
						},
					},
					userTagData{
						Names:       []string{"hey_root"},
						Description: "",
						Subtags:     []userTagData{},
					},
				},
			}

			err = tagDataEqual(&tdExpected, &tdGot)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// Get resulting tag tree from tag foo1 / b
		{
			resp, err := be.DoUserReq(
				"GET", "/tags/foo1/b", u1ID, nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			resp2, err := be.DoUserReq(
				"GET", fmt.Sprintf("/tags/%d", tagID_Foo1_b), u1ID, nil, true,
			)
			if err != nil {
				return errors.Trace(err)
			}

			var tdGot userTagData
			decoder := json.NewDecoder(resp.Body)
			decoder.Decode(&tdGot)

			var tdGot2 userTagData
			decoder = json.NewDecoder(resp2.Body)
			decoder.Decode(&tdGot2)

			tdExpected := userTagData{
				Names:       []string{"b"},
				Description: "",
				Subtags: []userTagData{
					userTagData{
						Names:       []string{"Привет"},
						Description: "",
						Subtags:     []userTagData{},
					},
				},
			}

			err = tagDataEqual(&tdExpected, &tdGot)
			if err != nil {
				return errors.Trace(err)
			}

			err = tagDataEqual(&tdExpected, &tdGot2)
			if err != nil {
				return errors.Trace(err)
			}
		}

		// --------- test updating tags ---------

		// Try to update tag foo1: make foo2 a primary name,
		// but do not change anything else
		err = updateTag(
			be, "/tags/foo1", u1ID, []string{"foo2", "foo1"}, nil, nil,
		)
		if err != nil {
			return errors.Trace(err)
		}

		err = expectSingleTag(be, "/tags/foo1", u1ID, &userTagData{
			Names:       []string{"foo2", "foo1"},
			Description: "my foo descr",
		})
		if err != nil {
			return errors.Trace(err)
		}

		// Try to update the description of the tag foo
		err = updateTag(
			be, "/tags/foo1", u1ID, nil, cptr.String("my updated foo descr"), nil,
		)
		if err != nil {
			return errors.Trace(err)
		}

		err = expectSingleTag(be, "/tags/foo1", u1ID, &userTagData{
			Names:       []string{"foo2", "foo1"},
			Description: "my updated foo descr",
		})
		if err != nil {
			return errors.Trace(err)
		}

		// Try to update the names AND the description of the tag foo
		err = updateTag(
			be, "/tags/foo1", u1ID,
			[]string{"name1", "name2"},
			cptr.String("my again updated foo descr"),
			nil,
		)
		if err != nil {
			return errors.Trace(err)
		}

		err = expectSingleTag(be, "/tags/name2", u1ID, &userTagData{
			Names:       []string{"name1", "name2"},
			Description: "my again updated foo descr",
		})
		if err != nil {
			return errors.Trace(err)
		}

		// And one more partial names update
		err = updateTag(
			be, "/tags/name2", u1ID,
			[]string{"name1", "name3"},
			nil,
			nil,
		)
		if err != nil {
			return errors.Trace(err)
		}

		err = expectSingleTag(be, "/tags/name1", u1ID, &userTagData{
			Names:       []string{"name1", "name3"},
			Description: "my again updated foo descr",
		})
		if err != nil {
			return errors.Trace(err)
		}

		// Try to update tag name to the already existing one (should fail)
		{
			resp, err := be.DoUserReq(
				"PUT", "/tags/name1", u1ID,
				H{"names": A{"name1", "foo3", "name3"}},
				false,
			)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(
				resp, http.StatusBadRequest, "Tag with the name \"foo3\" already exists",
			); err != nil {
				return errors.Trace(err)
			}
		}

		// Try to update tag of another user (should fail)
		{
			resp, err := be.DoReq(
				"PUT", fmt.Sprintf("/api/users/%d/tags", u1ID), u2Token,
				bytes.NewReader([]byte(`
				{"names": ["name1"]}
				`)),
				false,
			)
			if err != nil {
				return errors.Trace(err)
			}

			if err := expectErrorResp(
				resp, http.StatusForbidden, "forbidden",
			); err != nil {
				return errors.Trace(err)
			}
		}

		fmt.Println(tagID_Foo1, tagID_Foo3, tagID_Foo1_a, tagID_Foo1_b, tagID_Foo1_b_c)

		return nil
	})
}

func TestTagsByPattern(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		var u1ID int
		var u1Token string
		var err error

		if u1ID, u1Token, err = testutils.CreateTestUser(t, si, "test1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", u1Token)

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

		_, err = checkTagsGet(be, u1ID, "//////", true, []string{})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = checkTagsGet(be, u1ID, "tag7/8", true, []string{
			"/tag7/tag8",
		})
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
}

func TestTagsMoving(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		var u1ID int
		var u1Token string
		var err error

		// TODO: create test user without `si` (but via server instead)
		if u1ID, u1Token, err = testutils.CreateTestUser(t, si, "test1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", u1Token)

		err = perUserTestTagsMoving(t, si, be, u1ID, "test1", u1Token)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
}

func perUserTestTagsMoving(
	t *testing.T, si storage.Storage, be testBackend, userID int, username, token string,
) error {
	tagIDs, err := makeTestTagsHierarchy(be, userID)
	if err != nil {
		return errors.Trace(err)
	}

	bkmIDs, err := makeTestBookmarks(be, userID, tagIDs)
	if err != nil {
		return errors.Trace(err)
	}

	// get tagged with tag3
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag3ID}}, []int{
			bkmIDs.bkm3ID,
			bkmIDs.bkm4ID,
			bkmIDs.bkm5ID,
			bkmIDs.bkm6ID,
			bkmIDs.bkm2_5ID,
			bkmIDs.bkm4_5ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// get tagged with tag7
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag7ID}}, []int{
			bkmIDs.bkm7ID,
			bkmIDs.bkm8ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// Move tag5 under tag7; new tag hierarchy:
	// /
	// ├── tag1
	// │   └── tag3
	// │       └── tag4
	// ├── tag2
	// └── tag7
	//     ├── tag5
	//     │   └── tag6
	//     └── tag8
	err = updateTag(
		be, "/tags/tag1/tag3/tag5", userID, nil, nil, &tagIDs.tag7ID,
	)
	if err != nil {
		return errors.Trace(err)
	}

	err = si.CheckIntegrity()
	if err != nil {
		return errors.Trace(err)
	}

	// get tagged with tag3
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag3ID}}, []int{
			bkmIDs.bkm3ID,
			bkmIDs.bkm4ID,
			bkmIDs.bkm4_5ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// get tagged with tag7
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag7ID}}, []int{
			bkmIDs.bkm7ID,
			bkmIDs.bkm5ID,
			bkmIDs.bkm6ID,
			bkmIDs.bkm2_5ID,
			bkmIDs.bkm4_5ID,
			bkmIDs.bkm8ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// get tagged with tag5
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag5ID}}, []int{
			bkmIDs.bkm5ID,
			bkmIDs.bkm6ID,
			bkmIDs.bkm2_5ID,
			bkmIDs.bkm4_5ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}

func TestTagsDeletion(t *testing.T) {
	runWithRealDB(t, func(si storage.Storage, be testBackend) error {
		var u1ID int
		var u1Token string
		var err error

		// TODO: create test user without `si` (but via server instead)
		if u1ID, u1Token, err = testutils.CreateTestUser(t, si, "test1", "1@1.1"); err != nil {
			return errors.Trace(err)
		}
		be.UserCreated(u1ID, "test1", u1Token)

		err = perUserTestTagsDeletion(t, si, be, u1ID, "test1", u1Token)
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
}

func perUserTestTagsDeletion(
	t *testing.T, si storage.Storage, be testBackend, userID int, username, token string,
) error {
	tagIDs, err := makeTestTagsHierarchy(be, userID)
	if err != nil {
		return errors.Trace(err)
	}

	bkmIDs, err := makeTestBookmarks(be, userID, tagIDs)
	if err != nil {
		return errors.Trace(err)
	}

	if err := deleteTag(be, "/tags/tag1/tag3", userID); err != nil {
		return errors.Trace(err)
	}

	// get tagged with tag2
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag2ID}}, []int{
			bkmIDs.bkm2ID,
			bkmIDs.bkm2_5ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// get tagged with deleted tag5: should be nothing
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag5ID}}, []int{},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// get untagged: those tagged with the deleted tag3, etc, should not become
	// untagged, because they are still tagged with tag1
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{}}, []int{
			bkmIDs.bkm_untagged_ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// get tagged with tag1
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{tagIDs.tag1ID}}, []int{
			bkmIDs.bkm1ID,
			bkmIDs.bkm3ID,
			bkmIDs.bkm4ID,
			bkmIDs.bkm5ID,
			bkmIDs.bkm6ID,
			bkmIDs.bkm4_5ID,
			bkmIDs.bkm2_5ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// delete /tags/tag1, and make sure that there are new untagged
	// bookmarks
	if err := deleteTag(be, "/tags/tag1", userID); err != nil {
		return errors.Trace(err)
	}
	_, err = checkBkmGet(
		be, userID, &bkmGetArg{tagIDs: []int{}}, []int{
			bkmIDs.bkm_untagged_ID,
			bkmIDs.bkm1ID,
			bkmIDs.bkm3ID,
			bkmIDs.bkm4ID,
			bkmIDs.bkm5ID,
			bkmIDs.bkm6ID,
			bkmIDs.bkm4_5ID,
		},
	)
	if err != nil {
		return errors.Trace(err)
	}

	// test that deleting /tags (a root tag) should not be possible
	genResp, err := be.DoUserReq(
		"DELETE", "/tags", userID, nil, false,
	)
	if got, want := genResp.StatusCode, http.StatusBadRequest; got != want {
		return errors.Errorf("deleting root tag: want status code %d, got %d", want, got)
	}

	return nil
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

func expectSingleTag(
	be testBackend, url string, userID int, tdExpected *userTagData,
) error {
	resp, err := be.DoUserReq(
		"GET", fmt.Sprintf("%s?shape=single", url), userID, nil, true,
	)
	if err != nil {
		return errors.Trace(err)
	}

	var tdGot userTagData
	decoder := json.NewDecoder(resp.Body)
	decoder.Decode(&tdGot)

	err = tagDataEqual(tdExpected, &tdGot)
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
