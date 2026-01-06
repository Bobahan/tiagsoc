CREATE TABLE IF NOT EXISTS roles (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  level INT NOT NULL DEFAULT 0,
  description TEXT
);

INSERT INTO roles (name, level, description) VALUES ('user', 1, 'A user can create posts and comments');
INSERT INTO roles (name, level, description) VALUES ('moderator', 2, 'A moderator can update other users posts');
INSERT INTO roles (name, level, description) VALUES ('admin', 3, 'A admin can update and delete other users posts');