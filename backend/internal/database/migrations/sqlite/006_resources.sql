-- Workspace file tree ("resources"). The bytes live on disk under the
-- resources root (see docs/database.md); these rows are the source of truth
-- for the tree.
--
-- name_key is the case-folded name, so names are unique case-insensitively
-- among live siblings while name keeps the spelling the user typed.
-- A trashed subtree keeps its parent links; every row in it carries the id of
-- the row that was trashed (trashed_root_id), so restoring brings back exactly
-- what was trashed together. deleted_at marks a purged row: its bytes are gone
-- and the row stays as a tombstone for the activity log and synchronization.
CREATE TABLE resources (
    id              TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id    TEXT NOT NULL REFERENCES workspaces (id),
    parent_id       TEXT,
    kind            TEXT NOT NULL CHECK (kind IN ('folder', 'file')),
    name            TEXT NOT NULL CHECK (length(name) BETWEEN 1 AND 255),
    name_key        TEXT NOT NULL,
    mime_type       TEXT,
    size            INTEGER NOT NULL DEFAULT 0 CHECK (size >= 0),
    is_text         INTEGER NOT NULL DEFAULT 0 CHECK (is_text IN (0, 1)),
    revision        INTEGER NOT NULL DEFAULT 1 CHECK (revision >= 1),
    created_by      TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    updated_at      TEXT NOT NULL,
    trashed_at      TEXT,
    trashed_root_id TEXT,
    deleted_at      TEXT,
    UNIQUE (id, workspace_id),
    CHECK ((trashed_at IS NULL) = (trashed_root_id IS NULL)),
    CHECK (kind = 'file' OR (mime_type IS NULL AND size = 0 AND is_text = 0)),
    FOREIGN KEY (parent_id, workspace_id) REFERENCES resources (id, workspace_id),
    FOREIGN KEY (created_by, workspace_id) REFERENCES memberships (id, workspace_id)
) STRICT;

CREATE UNIQUE INDEX idx_resources_sibling_name
    ON resources (workspace_id, COALESCE(parent_id, ''), name_key)
    WHERE trashed_at IS NULL AND deleted_at IS NULL;
CREATE INDEX idx_resources_parent_id ON resources (parent_id);
CREATE INDEX idx_resources_trashed_root_id
    ON resources (trashed_root_id)
    WHERE trashed_root_id IS NOT NULL;
CREATE INDEX idx_resources_workspace_id_updated_at
    ON resources (workspace_id, updated_at);

-- Links between tasks and resources (files or folders).
CREATE TABLE task_resources (
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    task_id      TEXT NOT NULL,
    resource_id  TEXT NOT NULL,
    linked_by    TEXT NOT NULL,
    linked_at    TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    removed_at   TEXT,
    PRIMARY KEY (task_id, resource_id),
    FOREIGN KEY (task_id, workspace_id) REFERENCES tasks (id, workspace_id),
    FOREIGN KEY (resource_id, workspace_id) REFERENCES resources (id, workspace_id),
    FOREIGN KEY (linked_by, workspace_id) REFERENCES memberships (id, workspace_id)
) STRICT;

CREATE INDEX idx_task_resources_resource_id
    ON task_resources (resource_id)
    WHERE removed_at IS NULL;
CREATE INDEX idx_task_resources_workspace_id_updated_at
    ON task_resources (workspace_id, updated_at);

-- Who did what to which resource. Rows are only appended, except that an
-- open 'edited' row is extended (updated_at, details) while the same member
-- keeps editing the same file. task_id is set for link and unlink events.
CREATE TABLE resource_events (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    resource_id  TEXT NOT NULL,
    actor_id     TEXT NOT NULL,
    action       TEXT NOT NULL CHECK (action IN (
                     'created', 'uploaded', 'edited', 'renamed', 'moved', 'copied',
                     'trashed', 'restored', 'purged', 'linked', 'unlinked')),
    task_id      TEXT,
    details      TEXT NOT NULL DEFAULT '{}',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    FOREIGN KEY (resource_id, workspace_id) REFERENCES resources (id, workspace_id),
    FOREIGN KEY (actor_id, workspace_id) REFERENCES memberships (id, workspace_id),
    FOREIGN KEY (task_id, workspace_id) REFERENCES tasks (id, workspace_id)
) STRICT;

CREATE INDEX idx_resource_events_workspace_id_id ON resource_events (workspace_id, id);
CREATE INDEX idx_resource_events_resource_id_id ON resource_events (resource_id, id);
CREATE INDEX idx_resource_events_task_id ON resource_events (task_id) WHERE task_id IS NOT NULL;
CREATE INDEX idx_resource_events_workspace_id_updated_at
    ON resource_events (workspace_id, updated_at);

-- Per-user settings that follow the user to every device. Not scoped to a
-- workspace: a user carries them across all of their workspaces.
CREATE TABLE user_settings (
    user_id    TEXT NOT NULL REFERENCES users (id),
    key        TEXT NOT NULL CHECK (length(key) > 0),
    value      TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (user_id, key)
) STRICT;
