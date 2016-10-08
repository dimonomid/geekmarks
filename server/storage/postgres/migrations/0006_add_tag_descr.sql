-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "tags" ADD COLUMN "descr" TEXT NOT NULL DEFAULT '';

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE "tags" DROP COLUMN "descr";
