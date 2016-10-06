-- NULL parent is going to be used for root tag for each user

-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "tags" ALTER COLUMN "parent_id" DROP NOT NULL;

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE "tags" ALTER COLUMN "parent_id" SET NOT NULL;
