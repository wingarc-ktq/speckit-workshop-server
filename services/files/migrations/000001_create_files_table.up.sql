CREATE TABLE IF NOT EXISTS files (
    id           UUID PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    size         BIGINT NOT NULL,
    mime_type    VARCHAR(255) NOT NULL,
    description  VARCHAR(500) NOT NULL DEFAULT '',
    storage_key  VARCHAR(255) NOT NULL,
    tag_ids      UUID[] NOT NULL DEFAULT '{}',
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_files_name ON files (name);
CREATE INDEX IF NOT EXISTS idx_files_uploaded_at ON files (uploaded_at);
CREATE INDEX IF NOT EXISTS idx_files_tag_ids ON files USING GIN (tag_ids);
