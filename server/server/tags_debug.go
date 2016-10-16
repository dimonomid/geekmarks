package server

import (
	"database/sql"

	"dmitryfrank.com/geekmarks/server/storage"

	"github.com/juju/errors"
)

func (gm *GMServer) addTestTagsTree(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	{
		err = gm.si.Tx(func(tx *sql.Tx) error {
			parentTagID, err := gm.si.GetTagIDByPath(tx, gmr.SubjUser.ID, "")
			if err != nil {
				return errors.Trace(err)
			}

			_, err = gm.si.CreateTag(tx, &storage.TagData{
				OwnerID:     gmr.SubjUser.ID,
				ParentTagID: parentTagID,
				Names:       []string{"computer", "comp"},
				Description: "Everything related to computers",
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
					ParentTagID: parentTagID,
					Names:       []string{"programming"},
					Description: "",
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

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: parentTagID,
						Names:       []string{"c"},
						Description: "",
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
						ParentTagID: parentTagID,
						Names:       []string{"python"},
						Description: "",
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
						ParentTagID: parentTagID,
						Names:       []string{"ruby"},
						Description: "",
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
						ParentTagID: parentTagID,
						Names:       []string{"go", "golang"},
						Description: "",
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
					ParentTagID: parentTagID,
					Names:       []string{"linux", "gnu-linux"},
					Description: "",
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

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: parentTagID,
						Names:       []string{"udev"},
						Description: "",
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
						ParentTagID: parentTagID,
						Names:       []string{"systemd"},
						Description: "",
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
						ParentTagID: parentTagID,
						Names:       []string{"kernel"},
						Description: "",
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
				ParentTagID: parentTagID,
				Names:       []string{"life"},
				Description: "Everything NOT related to computers",
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
					ParentTagID: parentTagID,
					Names:       []string{"sport", "sports"},
					Description: "",
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

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: parentTagID,
						Names:       []string{"bike", "bicycle"},
						Description: "",
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

					_, err = gm.si.CreateTag(tx, &storage.TagData{
						OwnerID:     gmr.SubjUser.ID,
						ParentTagID: parentTagID,
						Names:       []string{"kayak"},
						Description: "",
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

	resp = map[string]string{"status": "ok"}

	return resp, nil
}
