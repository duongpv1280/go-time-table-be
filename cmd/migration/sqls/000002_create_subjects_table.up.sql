CREATE TABLE IF NOT EXISTS subjects (
    id         VARCHAR(36)  NOT NULL PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    created_at DATETIME     NOT NULL,
    updated_at DATETIME     NOT NULL
);
