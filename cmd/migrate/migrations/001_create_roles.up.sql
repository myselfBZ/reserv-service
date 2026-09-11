CREATE TABLE IF NOT EXISTS roles (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  level int NOT NULL DEFAULT 0,
  description TEXT
);


INSERT INTO
  roles (name, description, level)
VALUES
  (
    'user',
    'A user can order items and buy',
    1
  );


INSERT INTO
  roles (name, description, level)
VALUES
  (
    'admin',
    'An admin can add and edit products',
    10
  );
