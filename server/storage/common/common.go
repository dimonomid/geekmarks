// Copyright 2017 Dmitry Frank <mail@dmitryfrank.com>
// Licensed under the BSD, see LICENSE file for details.

package common // import "dmitryfrank.com/geekmarks/server/storage/common"

import (
	"flag"
	"os"

	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/storage/postgres"

	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

var (
	dbType = flag.String("geekmarks.dbtype", "postgres",
		"Database type. So far, only postgres is supported.")
	postgresURL = flag.String("geekmarks.postgres.url", "",
		"Data source name pointing to the Postgres database. Alternatively, can be "+
			"given in an environment variable GM_POSTGRES_URL.")
)

func CreateStorage() (storage.Storage, error) {
	switch *dbType {
	case "postgres":
		pgURL := *postgresURL
		if pgURL == "" {
			pgURL = os.Getenv("GM_POSTGRES_URL")
		}
		return postgres.New(pgURL)
	default:
		return nil, errors.Errorf("Invalid database type: %q", *dbType)
	}
}
