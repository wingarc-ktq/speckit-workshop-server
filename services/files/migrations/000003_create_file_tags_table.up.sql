CREATE TABLE file_tags (
    file_id UUID NOT NULL REFERENCES files (id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (file_id, tag_id)
);

CREATE INDEX idx_file_tags_tag_id ON file_tags (tag_id);
