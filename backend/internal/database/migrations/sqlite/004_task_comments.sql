-- Comments left on a task. author_id is the membership that wrote it; only
-- the author may edit or delete a comment (enforced in the service layer).
CREATE TABLE task_comments (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    task_id      TEXT NOT NULL,
    author_id    TEXT NOT NULL,
    body         TEXT NOT NULL CHECK (length(body) > 0),
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT,
    UNIQUE (id, workspace_id),
    FOREIGN KEY (task_id, workspace_id) REFERENCES tasks (id, workspace_id),
    FOREIGN KEY (author_id, workspace_id) REFERENCES memberships (id, workspace_id)
) STRICT;

CREATE INDEX idx_task_comments_task_id
    ON task_comments (task_id, created_at)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_task_comments_workspace_id_updated_at
    ON task_comments (workspace_id, updated_at);
