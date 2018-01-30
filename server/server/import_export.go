// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package server

import (
	"database/sql"

	"dmitryfrank.com/geekmarks/server/storage"
	"github.com/juju/errors"
)

type userDataDump struct {
	StorageType storage.StorageType `json:"storage_type"`
	DumpVersion string              `json:"dump_version"`
	Data        interface{}         `json:"data"`
}

func (gm *GMServer) userDataExport(gmr *GMRequest) (resp interface{}, err error) {
	err = gm.authorizeOperation(gmr.Caller, &authzArgs{OwnerID: gmr.SubjUser.ID})
	if err != nil {
		return nil, errors.Trace(err)
	}

	dump := userDataDump{}

	if err := gm.si.TxOpt(
		storage.TxILevelRepeatableRead, storage.TxModeReadOnly,
		func(tx *sql.Tx) error {
			storageDump, err := gm.si.Export(tx, gmr.SubjUser.ID)
			if err != nil {
				return errors.Trace(err)
			}

			dump.StorageType = storageDump.StorageType
			dump.DumpVersion = storageDump.DumpVersion
			dump.Data = storageDump.Data

			return nil
		},
	); err != nil {
		return nil, errors.Trace(err)
	}

	return dump, nil
}
