package common

import (
	"flag"

	"dmitryfrank.com/geekmarks/server/storage"
	"dmitryfrank.com/geekmarks/server/storage/postgres"

	"github.com/juju/errors"
	_ "github.com/lib/pq"
)

var (
	dbType = flag.String("geekmarks.dbtype", "postgres",
		"Database type. So far, only postgres is supported.")
	postgresURL = flag.String("geekmarks.postgres.url", "",
		"Data source name pointing to the Postgres database.")
)

func CreateStorage() (storage.Storage, error) {
	switch *dbType {
	case "postgres":
		return postgres.New(*postgresURL)
	default:
		return nil, errors.Errorf("Invalid database type: %q", *dbType)
	}
}
