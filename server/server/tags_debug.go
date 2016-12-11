package server

import (
	"database/sql"
	"fmt"

	"dmitryfrank.com/geekmarks/server/cptr"
	"dmitryfrank.com/geekmarks/server/storage"

	"github.com/juju/errors"
)

const (
	skipBkm = "skip_bookmarks"
)

func (gm *GMServer) addBookmark(
	gmr *GMRequest, tx *sql.Tx,
	title, comment string,
	tagIDs []int,
) (bkmID int, err error) {
	for _, v := range tagIDs {
		if v == 0 {
			fmt.Printf("skipping creation of bookmark %s\n", title)
			return 0, nil
		}
	}

	if gmr.FormValue(skipBkm) == "" {
		bkmID, err := gm.si.CreateBookmark(tx, &storage.BookmarkData{
			OwnerID: gmr.SubjUser.ID,
			URL:     fmt.Sprintf("https://google.com?q=%s", title),
			Title:   title,
			Comment: comment,
		})
		if err != nil {
			return 0, errors.Trace(err)
		}

		err = gm.si.SetTaggings(tx, bkmID, tagIDs, storage.TaggingModeLeafs)
		if err != nil {
			return 0, errors.Trace(err)
		}
	}

	return bkmID, nil
}

func (gm *GMServer) addTestTagsTree(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	var tagIDProgC, tagIDUdev, tagIDKernel, tagIDProgGo, tagIDBike, tagIDKayak int

	{
		err = gm.si.Tx(func(tx *sql.Tx) error {
			parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "")
			if err != nil {
				return errors.Trace(err)
			}

			_, err = gm.si.CreateTag(tx, &storage.TagData{
				OwnerID:     gmr.SubjUser.ID,
				ParentTagID: cptr.Int(parentTagID),
				Names:       []string{"computer", "comp"},
				Description: cptr.String("Everything related to computers"),
			})
			if err != nil {
				return errors.Trace(err)
			}
			return nil
		})
		if err != nil {
			//return nil, errors.Trace(err)
		}

		{
			err = gm.si.Tx(func(tx *sql.Tx) error {
				parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer")
				if err != nil {
					return errors.Trace(err)
				}

				_, err = gm.si.CreateTag(tx, &storage.TagData{
					OwnerID:     gmr.SubjUser.ID,
					ParentTagID: cptr.Int(parentTagID),
					Names:       []string{"programming"},
					Description: cptr.String(""),
				})
				if err != nil {
					return errors.Trace(err)
				}
				return nil
			})
			if err != nil {
				//return nil, errors.Trace(err)
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/programming")
					if err != nil {
						return errors.Trace(err)
					}

					tagIDProgC, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"c"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}

					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/programming")
					if err != nil {
						return errors.Trace(err)
					}

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"python"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/programming")
					if err != nil {
						return errors.Trace(err)
					}

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"javascript"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/programming")
					if err != nil {
						return errors.Trace(err)
					}

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"ruby"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/programming")
					if err != nil {
						return errors.Trace(err)
					}

					tagIDProgGo, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"go", "golang"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}
		}

		{
			err = gm.si.Tx(func(tx *sql.Tx) error {
				parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer")
				if err != nil {
					return errors.Trace(err)
				}

				_, err = gm.si.CreateTag(tx, &storage.TagData{
					OwnerID:     gmr.SubjUser.ID,
					ParentTagID: cptr.Int(parentTagID),
					Names:       []string{"linux", "gnu-linux"},
					Description: cptr.String(""),
				})
				if err != nil {
					return errors.Trace(err)
				}
				return nil
			})
			if err != nil {
				//return nil, errors.Trace(err)
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/linux")
					if err != nil {
						return errors.Trace(err)
					}

					tagIDUdev, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"udev"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/linux")
					if err != nil {
						return errors.Trace(err)
					}

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"systemd"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "computer/linux")
					if err != nil {
						return errors.Trace(err)
					}

					tagIDKernel, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"kernel"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}
		}

	}

	{
		err = gm.si.Tx(func(tx *sql.Tx) error {
			parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "")
			if err != nil {
				return errors.Trace(err)
			}

			_, err = gm.si.CreateTag(tx, &storage.TagData{
				OwnerID:     gmr.SubjUser.ID,
				ParentTagID: cptr.Int(parentTagID),
				Names:       []string{"life"},
				Description: cptr.String("Everything NOT related to computers"),
			})
			if err != nil {
				return errors.Trace(err)
			}
			return nil
		})
		if err != nil {
			//return nil, errors.Trace(err)
		}

		{
			err = gm.si.Tx(func(tx *sql.Tx) error {
				parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "life")
				if err != nil {
					return errors.Trace(err)
				}

				_, err = gm.si.CreateTag(tx, &storage.TagData{
					OwnerID:     gmr.SubjUser.ID,
					ParentTagID: cptr.Int(parentTagID),
					Names:       []string{"sport", "sports"},
					Description: cptr.String(""),
				})
				if err != nil {
					return errors.Trace(err)
				}
				return nil
			})
			if err != nil {
				//return nil, errors.Trace(err)
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "life/sports")
					if err != nil {
						return errors.Trace(err)
					}

					tagIDBike, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"bike", "bicycle"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}

			{
				err = gm.si.Tx(func(tx *sql.Tx) error {
					parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "life/sports")
					if err != nil {
						return errors.Trace(err)
					}

					tagIDKayak, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: cptr.Int(parentTagID),
						Names:       []string{"kayak"},
						Description: cptr.String(""),
					})
					if err != nil {
						return errors.Trace(err)
					}
					return nil
				})
				if err != nil {
					//return nil, errors.Trace(err)
				}
			}
		}

	}

	err = gm.si.Tx(func(tx *sql.Tx) error {
		var err error
		_, err = gm.addBookmark(gmr, tx, "Something about C", "", []int{tagIDProgC})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about Udev and C", "", []int{tagIDProgC, tagIDUdev})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about Udev", "", []int{tagIDUdev})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about Go 1", "", []int{tagIDProgGo})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about Go 2", "", []int{tagIDProgGo})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about kernel and C", "", []int{tagIDProgC, tagIDKernel})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about bicycles", "", []int{tagIDBike})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about kayak 1", "", []int{tagIDKayak})
		if err != nil {
			return errors.Trace(err)
		}

		_, err = gm.addBookmark(gmr, tx, "Something about kayak 2", "", []int{tagIDKayak})
		if err != nil {
			return errors.Trace(err)
		}

		return nil
	})
	if err != nil {
		return nil, errors.Trace(err)
	}

	resp = map[string]string{"status": "ok"}

	return resp, nil
}
