CREATE TYPE tag_color AS ENUM ('blue', 'red', 'yellow', 'green', 'purple', 'orange', 'gray');

CREATE TABLE tags (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color tag_color NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tags_name ON tags (name);
