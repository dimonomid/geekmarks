-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied

-- +migrate StatementBegin
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
-- +migrate StatementEnd

CREATE TRIGGER check_dup_null BEFORE INSERT OR UPDATE ON tags
    FOR EACH ROW EXECUTE PROCEDURE check_dup_null();

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
