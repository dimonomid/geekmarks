-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "bookmarks" DROP COLUMN "owner_id" RESTRICT;

ALTER TABLE "taggables" ADD COLUMN "owner_id" INTEGER NOT NULL;

ALTER TABLE "taggables" ADD CONSTRAINT "taggables_owner_id_fkey"
  FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;
-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE "taggables" DROP COLUMN "owner_id" RESTRICT;

ALTER TABLE "bookmarks" ADD COLUMN "owner_id" INTEGER NOT NULL;

ALTER TABLE "bookmarks" ADD CONSTRAINT "bookmarks_owner_id_fkey"
  FOREIGN KEY (owner_id) REFERENCES users(id);
