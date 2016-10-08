-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "tags" DROP CONSTRAINT "tags_parent_id_fkey";

ALTER TABLE "tags" ADD
  FOREIGN KEY (parent_id) REFERENCES tags(id) ON DELETE CASCADE;

ALTER TABLE "tags" DROP CONSTRAINT "tags_owner_id_fkey";

ALTER TABLE "tags" ADD
  FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE "tag_names" DROP CONSTRAINT "tag_names_tag_id_fkey";

ALTER TABLE "tag_names" ADD
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;

ALTER TABLE "bookmarks" DROP CONSTRAINT "bookmarks_id_fkey";

ALTER TABLE "bookmarks" ADD
  FOREIGN KEY (id) REFERENCES taggables(id) ON DELETE CASCADE;

ALTER TABLE "taggings" DROP CONSTRAINT "taggings_taggable_id_fkey";

ALTER TABLE "taggings" ADD
  FOREIGN KEY (taggable_id) REFERENCES taggables(id) ON DELETE CASCADE;

ALTER TABLE "taggings" DROP CONSTRAINT "taggings_tag_id_fkey";

ALTER TABLE "taggings" ADD
  FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE;

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
