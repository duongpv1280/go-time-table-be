CREATE TABLE IF NOT EXISTS users (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    email      VARCHAR(255) NOT NULL UNIQUE,
    name       VARCHAR(255) NOT NULL,
    created_at DATETIME     NOT NULL,
    updated_at DATETIME     NOT NULL
);
