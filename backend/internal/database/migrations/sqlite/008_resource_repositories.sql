-- A resource folder whose content is a GitHub repository. The folder row
-- stays in the resource tree (it can be renamed, moved, trashed and linked to
-- tasks like any folder); this row says which repository fills it and how it
-- is fetched. The files themselves are not resource rows: every device
-- fetches its own read-only copy into RESOURCES_PATH/<workspace>/.yap-repos/.
-- The row lives and dies with its resource row, so it has no deleted_at.
CREATE TABLE resource_repositories (
    resource_id  TEXT NOT NULL PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    github_id    INTEGER NOT NULL CHECK (github_id > 0),
    full_name    TEXT NOT NULL CHECK (length(full_name) BETWEEN 3 AND 200),
    html_url     TEXT NOT NULL,
    branch       TEXT NOT NULL CHECK (length(branch) BETWEEN 1 AND 255),
    is_private   INTEGER NOT NULL CHECK (is_private IN (0, 1)),
    method       TEXT NOT NULL CHECK (method IN ('zip', 'git')),
    added_by     TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    UNIQUE (resource_id, workspace_id),
    FOREIGN KEY (resource_id, workspace_id) REFERENCES resources (id, workspace_id),
    FOREIGN KEY (added_by, workspace_id) REFERENCES memberships (id, workspace_id)
) STRICT;

CREATE INDEX idx_resource_repositories_workspace_id_updated_at
    ON resource_repositories (workspace_id, updated_at);

-- This device's copy of a repository. Local only, never synchronized.
-- commit_sha and fetched_at describe the copy on disk; checked_at is the last
-- time GitHub answered; attempted_at the last fetch or update attempt, and
-- error why it failed (NULL after a success).
CREATE TABLE repository_checkouts (
    resource_id TEXT NOT NULL PRIMARY KEY REFERENCES resource_repositories (resource_id),
    state       TEXT NOT NULL CHECK (state IN ('pending', 'fetching', 'ready')),
    commit_sha  TEXT,
    fetched_at  TEXT,
    checked_at  TEXT,
    attempted_at TEXT,
    error       TEXT CHECK (error IN ('offline', 'unauthorized', 'not_found', 'too_large', 'git_missing', 'failed')),
    updated_at  TEXT NOT NULL,
    -- fetching may be an update of a copy that is already there.
    CHECK (state != 'ready' OR commit_sha IS NOT NULL),
    CHECK (state != 'pending' OR commit_sha IS NULL)
) STRICT;
