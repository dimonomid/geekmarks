package storage

//go:generate go-bindata -nocompress -modtime 1 -mode 420 -pkg storage migrations

import (
	"database/sql"
	"flag"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/juju/errors"
	"github.com/rubenv/sql-migrate"
)

var (
	dbType = flag.String("geekmarks.dbtype", "mysql",
		"Database type. So far, only mysql is supported.")
	mysqlDSN = flag.String("geekmarks.mysql.dsn", "",
		"Data source name pointing to the MySQL database. "+
			"See https://github.com/go-sql-driver/mysql#dsn-data-source-name for the format.")

	db *sql.DB
)

func open() error {
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
		fmt.Printf("op my %v\n", db)
	default:
		return errors.Errorf("unknown dbtype: %q", *dbType)
	}

	return nil
}

func applyMigrations() error {
	migrations := &migrate.AssetMigrationSource{
		Asset:    Asset,
		AssetDir: AssetDir,
		Dir:      "migrations",
	}

	n, err := migrate.Exec(db, "mysql", migrations, migrate.Up)
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

func Initialize() error {
	err := open()
	if err != nil {
		return errors.Trace(err)
	}

	err = applyMigrations()
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
