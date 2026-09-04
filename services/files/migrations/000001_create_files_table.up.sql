CREATE TABLE files (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    size BIGINT NOT NULL CHECK (size >= 0 AND size <= 10485760),
    mime_type TEXT NOT NULL,
    description TEXT,
    storage_key TEXT NOT NULL UNIQUE,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_files_name ON files (name);
CREATE INDEX idx_files_uploaded_at ON files (uploaded_at DESC);
