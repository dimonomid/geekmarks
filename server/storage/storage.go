package storage

//go:generate go-bindata -nocompress -modtime 1 -mode 420 -pkg storage migrations

import (
	"database/sql"
	"flag"

	"dmitryfrank.com/geekmarks/server/cptr"
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

func createTestUsers() error {
	err := Tx(func(tx *sql.Tx) error {

		_, err := GetUser(tx, &GetUserArgs{
			ID: cptr.Int(1),
		})

		if err != nil {
			if errors.Cause(err) == ErrUserDoesNotExist {
				glog.Infof("Creating test user: alice")
				_, err := CreateUser(tx, &UserData{
					Username: "alice",
					Password: "alice",
					Email:    "alice@domain.com",
				})
				if err != nil {
					return errors.Trace(err)
				}
			} else {
				return errors.Trace(err)
			}
		}

		_, err = GetUser(tx, &GetUserArgs{
			ID: cptr.Int(2),
		})

		if err != nil {
			if errors.Cause(err) == ErrUserDoesNotExist {
				glog.Infof("Creating test user: bob")
				_, err := CreateUser(tx, &UserData{
					Username: "bob",
					Password: "bob",
					Email:    "bob@domain.com",
				})
				if err != nil {
					return errors.Trace(err)
				}
			} else {
				return errors.Trace(err)
			}
		}

		return nil
	})

	return errors.Trace(err)
}

func applyMigrations() error {
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

func Initialize() error {
	err := open()
	if err != nil {
		return errors.Trace(err)
	}

	err = applyMigrations()
	if err != nil {
		return errors.Trace(err)
	}

	err = createTestUsers()
	if err != nil {
		return errors.Trace(err)
	}

	return nil
}
