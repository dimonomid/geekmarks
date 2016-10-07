package storage

//go:generate go-bindata -nocompress -modtime 1 -mode 420 -pkg storage migrations

import (
	"database/sql"
	"flag"

	"github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/juju/errors"
	_ "github.com/lib/pq"
	"github.com/rubenv/sql-migrate"
)

var (
	dbType = flag.String("geekmarks.dbtype", "postgres",
		"Database type. So far, only mysql is supported.")
	mysqlDSN = flag.String("geekmarks.mysql.dsn", "",
		"Data source name pointing to the MySQL database. "+
			"See https://github.com/go-sql-driver/mysql#dsn-data-source-name for the format.")
	postgresURL = flag.String("geekmarks.postgres.url", "",
		"Data source name pointing to the Postgres database.")

	db *sql.DB
)

func InitConnection() error {
	switch *dbType {
	case "mysql":
		dsn, err := mysql.ParseDSN(*mysqlDSN)
		if err != nil {
			return errors.Trace(err)
		}
		db, err = sql.Open("mysql", dsn.FormatDSN())
		if err != nil {
			return errors.Trace(err)
		}
	case "postgres":
		var err error
		db, err = sql.Open("postgres", *postgresURL)
		if err != nil {
			return errors.Trace(err)
		}
	default:
		return errors.Errorf("unknown dbtype: %q", *dbType)
	}

	return nil
}

func ApplyMigrations() error {
	migrations := &migrate.AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      "migrations",
	}

	n, err := migrate.Exec(db, *dbType, migrations, migrate.Up)
	if n == 0 {
		glog.Infof("No migrations applied")
	} else {
		glog.Infof("Applied %d migrations!", n)
	}
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
