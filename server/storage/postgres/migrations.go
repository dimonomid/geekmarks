package postgres

import (
	"database/sql"

	"github.com/juju/errors"

	"dmitryfrank.com/geekmarks/server/dfmigrate"
)

func initMigrations() (*dfmigrate.Migrations, error) {
	mig := &dfmigrate.Migrations{}
	var err error

	// 001: Initial structure {{{
	err = mig.AddMigration(
		1, "Initial structure",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			if _, err := tx.Exec(`
				CREATE TABLE users (
					id SERIAL NOT NULL PRIMARY KEY,
					username VARCHAR(50),
					password VARCHAR(100),
					email VARCHAR(50)
				)
				`); err != nil {
				return errors.Trace(err)
			}

			if _, err := tx.Exec(`
				CREATE TABLE tags (
					id SERIAL NOT NULL PRIMARY KEY,
					parent_id INTEGER NOT NULL,
					owner_id INTEGER NOT NULL,
					FOREIGN KEY (parent_id) REFERENCES tags(id),
					FOREIGN KEY (owner_id) REFERENCES users(id)
				)
				`); err != nil {
				return errors.Trace(err)
			}

			if _, err := tx.Exec(`
				CREATE TABLE tag_names (
					id SERIAL NOT NULL PRIMARY KEY,
					tag_id INTEGER NOT NULL,
					name VARCHAR(30) NOT NULL,
					FOREIGN KEY (tag_id) REFERENCES tags(id)
				)
      `); err != nil {
				return errors.Trace(err)
			}

			if _, err := tx.Exec(`
				CREATE TABLE taggables (
					id SERIAL NOT NULL PRIMARY KEY,
					"type" VARCHAR(30) NOT NULL
				)
				`); err != nil {
				return errors.Trace(err)
			}

			if _, err := tx.Exec(`
				CREATE TABLE bookmarks (
					id SERIAL NOT NULL PRIMARY KEY,
					url TEXT NOT NULL,
					comment TEXT NOT NULL,
					owner_id INTEGER NOT NULL,
					FOREIGN KEY (id) REFERENCES taggables(id),
					FOREIGN KEY (owner_id) REFERENCES users(id)
				)
				`); err != nil {
				return errors.Trace(err)
			}

			if _, err := tx.Exec(`
				CREATE TABLE taggings (
					id SERIAL NOT NULL PRIMARY KEY,
					taggable_id INTEGER NOT NULL,
					tag_id INTEGER NOT NULL,
					FOREIGN KEY (taggable_id) REFERENCES taggables(id),
					FOREIGN KEY (tag_id) REFERENCES tags(id)
				)
				`); err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			if _, err := tx.Exec(`DROP TABLE users`); err != nil {
				return errors.Trace(err)
			}
			if _, err := tx.Exec(`DROP TABLE tags`); err != nil {
				return errors.Trace(err)
			}
			if _, err := tx.Exec(`DROP TABLE tag_names`); err != nil {
				return errors.Trace(err)
			}
			if _, err := tx.Exec(`DROP TABLE bookmarks`); err != nil {
				return errors.Trace(err)
			}
			if _, err := tx.Exec(`DROP TABLE taggables`); err != nil {
				return errors.Trace(err)
			}
			if _, err := tx.Exec(`DROP TABLE taggings`); err != nil {
				return errors.Trace(err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 002: Drop parent_id NOT NULL {{{
	// NULL parent is going to be used for root tag for each user
	err = mig.AddMigration(
		2, "Drop parent_id NOT NULL",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				ALTER TABLE "tags" ALTER COLUMN "parent_id" DROP NOT NULL
			`)
			return errors.Trace(err)
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			_, err := tx.Exec(`
				ALTER TABLE "tags" ALTER COLUMN "parent_id" SET NOT NULL
			`)
			return errors.Trace(err)
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 003: Move owner_id to taggables {{{
	err = mig.AddMigration(
		3, "Move owner_id to taggables",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
				ALTER TABLE "bookmarks" DROP COLUMN "owner_id" RESTRICT
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
				ALTER TABLE "taggables" ADD COLUMN "owner_id" INTEGER NOT NULL
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
				ALTER TABLE "taggables" ADD CONSTRAINT "taggables_owner_id_fkey"
					FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
				ALTER TABLE "taggables" DROP COLUMN "owner_id" RESTRICT
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
				ALTER TABLE "bookmarks" ADD COLUMN "owner_id" INTEGER NOT NULL
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
				ALTER TABLE "bookmarks" ADD CONSTRAINT "bookmarks_owner_id_fkey"
					FOREIGN KEY (owner_id) REFERENCES users(id)
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 004: Add ON DELETE {{{
	err = mig.AddMigration(
		4, "Add ON DELETE",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
ALTER TABLE "tags" DROP CONSTRAINT "tags_parent_id_fkey"
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tags" ADD
  FOREIGN KEY (parent_id) REFERENCES tags(id) ON DELETE CASCADE;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tags" DROP CONSTRAINT "tags_owner_id_fkey";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tags" ADD
  FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tag_names" DROP CONSTRAINT "tag_names_tag_id_fkey";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tag_names" ADD
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "bookmarks" DROP CONSTRAINT "bookmarks_id_fkey";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "bookmarks" ADD
  FOREIGN KEY (id) REFERENCES taggables(id) ON DELETE CASCADE;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "taggings" DROP CONSTRAINT "taggings_taggable_id_fkey";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "taggings" ADD
  FOREIGN KEY (taggable_id) REFERENCES taggables(id) ON DELETE CASCADE;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "taggings" DROP CONSTRAINT "taggings_tag_id_fkey";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "taggings" ADD
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			//TODO
			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 005: Use natural key for tag names {{{
	err = mig.AddMigration(
		5, "Use natural key for tag names",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
ALTER TABLE "tag_names" DROP CONSTRAINT "tag_names_pkey";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tag_names" ADD PRIMARY KEY (tag_id, name);
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tag_names" DROP COLUMN "id" RESTRICT;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
ALTER TABLE "tag_names" DROP CONSTRAINT "tag_names_pkey";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tag_names" ADD COLUMN "id" INTEGER NOT NULL;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE "tag_names" ADD PRIMARY KEY (id);
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 006: Add tag description {{{
	err = mig.AddMigration(
		6, "Add tag description",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
ALTER TABLE "tags" ADD COLUMN "descr" TEXT NOT NULL DEFAULT '';
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
ALTER TABLE "tags" DROP COLUMN "descr";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 007: Allow only one tag with NULL parent {{{
	err = mig.AddMigration(
		7, "Allow only one tag with NULL parent",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
CREATE OR REPLACE FUNCTION check_dup_null() RETURNS trigger AS $check_dup_null$
  DECLARE
    cnt INTEGER;
  BEGIN
    -- Check that empname and salary are given
    IF NEW.parent_id IS NULL THEN
      SELECT COUNT(id) INTO cnt FROM "tags" WHERE "parent_id" IS NULL and "owner_id" = NEW.owner_id;
      IF cnt > 0 THEN
        RAISE EXCEPTION 'duplicate tag with null parent_id for this owner_id';
      END IF;
    END IF;
    RETURN NEW;
  END;
$check_dup_null$ LANGUAGE plpgsql;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
CREATE TRIGGER check_dup_null BEFORE INSERT OR UPDATE ON tags
    FOR EACH ROW EXECUTE PROCEDURE check_dup_null();
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			// TODO
			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 008: Use enum for taggable type {{{
	err = mig.AddMigration(
		8, "Use enum for taggable type",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
CREATE TYPE taggable_type AS ENUM ('bookmark');
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE taggables DROP COLUMN "type";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE taggables ADD COLUMN "type" taggable_type NOT NULL;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			_, err = tx.Exec(`
ALTER TABLE taggables DROP COLUMN "type";
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE taggables ADD COLUMN "type" VARCHAR(30) NOT NULL;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
DROP TYPE taggable_type;
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}
	// 009: Add auto-updating timestamps to taggables {{{
	err = mig.AddMigration(
		9, "Add auto-updating timestamps to taggables",

		// ---------- UP ----------
		func(tx *sql.Tx) error {
			var err error
			_, err = tx.Exec(`
ALTER TABLE taggables
  ADD COLUMN "created_ts" TIMESTAMPTZ NOT NULL
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
ALTER TABLE taggables
  ADD COLUMN "updated_ts" TIMESTAMPTZ NOT NULL
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
    CREATE OR REPLACE FUNCTION set_created_ts () RETURNS trigger AS'
    BEGIN
        NEW.created_ts = NOW();
        RETURN NEW;
    END;
    'LANGUAGE 'plpgsql' IMMUTABLE
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
    CREATE OR REPLACE FUNCTION set_updated_ts () RETURNS trigger AS'
    BEGIN
        NEW.updated_ts = NOW();
        RETURN NEW;
    END;
    'LANGUAGE 'plpgsql' IMMUTABLE
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
    CREATE TRIGGER "trg_set_created_ts" BEFORE INSERT
    ON taggables FOR EACH ROW
    EXECUTE PROCEDURE set_created_ts();
			`)
			if err != nil {
				return errors.Trace(err)
			}

			_, err = tx.Exec(`
    CREATE TRIGGER "trg_set_updated_ts" BEFORE INSERT OR UPDATE
    ON taggables FOR EACH ROW
    EXECUTE PROCEDURE set_updated_ts();
			`)
			if err != nil {
				return errors.Trace(err)
			}

			return nil
		},

		// ---------- DOWN ----------
		func(tx *sql.Tx) error {
			//TODO
			return nil
		},
	)
	if err != nil {
		return nil, errors.Trace(err)
	}
	// }}}

	return mig, nil
}
