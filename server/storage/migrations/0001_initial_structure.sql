-- +migrate Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE users (
  id INTEGER NOT NULL AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(50),
  password VARCHAR(100),
  email VARCHAR(50)
);

CREATE TABLE tags (
  id INTEGER NOT NULL AUTO_INCREMENT PRIMARY KEY,
  parent_id INTEGER NOT NULL,
  owner_id INTEGER NOT NULL,
  FOREIGN KEY (parent_id) REFERENCES tags(id),
  FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE tag_names (
  id INTEGER NOT NULL AUTO_INCREMENT PRIMARY KEY,
  tag_id INTEGER NOT NULL,
  name VARCHAR(30) NOT NULL,
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE taggables (
  id INTEGER NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `type` VARCHAR(30) NOT NULL
);

CREATE TABLE bookmarks (
  id INTEGER NOT NULL PRIMARY KEY,
  url TEXT NOT NULL,
  comment TEXT NOT NULL,
  owner_id INTEGER NOT NULL,
  FOREIGN KEY (id) REFERENCES taggables(id),
  FOREIGN KEY (owner_id) REFERENCES users(id)
);

CREATE TABLE taggings (
  id INTEGER NOT NULL AUTO_INCREMENT PRIMARY KEY,
  taggable_id INTEGER NOT NULL,
  tag_id INTEGER NOT NULL,
  FOREIGN KEY (taggable_id) REFERENCES taggables(id),
  FOREIGN KEY (tag_id) REFERENCES tags(id)
);

-- +migrate Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE users;
DROP TABLE tags;
DROP TABLE tag_names;
DROP TABLE bookmarks;
DROP TABLE taggables;
DROP TABLE taggings;
