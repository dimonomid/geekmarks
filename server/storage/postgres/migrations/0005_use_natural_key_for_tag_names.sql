-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "tag_names" DROP CONSTRAINT "tag_names_pkey";

ALTER TABLE "tag_names" ADD PRIMARY KEY (tag_id, name);

ALTER TABLE "tag_names" DROP COLUMN "id" RESTRICT;

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE "tag_names" DROP CONSTRAINT "tag_names_pkey";

ALTER TABLE "tag_names" ADD COLUMN "id" INTEGER NOT NULL;

ALTER TABLE "tag_names" ADD PRIMARY KEY (id);
