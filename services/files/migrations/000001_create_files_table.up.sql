CREATE TABLE IF NOT EXISTS files (
    id            UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL,
    name          VARCHAR(255) NOT NULL,
    size          BIGINT NOT NULL CHECK (size > 0),
    mime_type     VARCHAR(100) NOT NULL,
    description   TEXT,
    uploaded_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_files_owner_uploaded_at
    ON files (owner_user_id, uploaded_at DESC);

CREATE INDEX IF NOT EXISTS idx_files_name
    ON files (name);
