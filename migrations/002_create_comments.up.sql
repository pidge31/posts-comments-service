CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY,
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id UUID NULL REFERENCES comments(id) ON DELETE CASCADE,
    author_id TEXT NOT NULL,
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,

    CONSTRAINT comments_text_length CHECK (char_length(text) <= 2000)
);

CREATE INDEX IF NOT EXISTS idx_comments_post_parent_created_id
ON comments (post_id, parent_id, created_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_comments_post_created_id
ON comments (post_id, created_at ASC, id ASC);