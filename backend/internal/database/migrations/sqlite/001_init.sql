CREATE TABLE users (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    display_name TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
) STRICT;

CREATE TABLE workspaces (
    id         TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    kind       TEXT NOT NULL CHECK (kind IN ('personal', 'team')),
    name       TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
) STRICT;

CREATE TABLE memberships (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    user_id      TEXT NOT NULL REFERENCES users (id),
    role         TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'member', 'viewer')),
    joined_at    TEXT NOT NULL,
    left_at      TEXT,
    updated_at   TEXT NOT NULL,
    UNIQUE (workspace_id, user_id),
    UNIQUE (id, workspace_id)
) STRICT;

CREATE INDEX idx_memberships_user_id ON memberships (user_id);

CREATE TABLE teams (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    name         TEXT NOT NULL,
    color        TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT,
    UNIQUE (id, workspace_id)
) STRICT;

CREATE UNIQUE INDEX idx_teams_workspace_id_name
    ON teams (workspace_id, name)
    WHERE deleted_at IS NULL;

CREATE TABLE projects (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    name         TEXT NOT NULL,
    description  TEXT,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    archived_at  TEXT,
    deleted_at   TEXT,
    UNIQUE (id, workspace_id)
) STRICT;

CREATE INDEX idx_projects_workspace_id
    ON projects (workspace_id)
    WHERE deleted_at IS NULL;

CREATE TABLE statuses (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    project_id   TEXT NOT NULL,
    title        TEXT NOT NULL,
    description  TEXT,
    color        TEXT NOT NULL,
    category     TEXT NOT NULL CHECK (category IN ('todo', 'in_progress', 'done')),
    position     TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT,
    UNIQUE (id, project_id),
    FOREIGN KEY (project_id, workspace_id) REFERENCES projects (id, workspace_id)
) STRICT;

CREATE INDEX idx_statuses_project_id_position ON statuses (project_id, position);

CREATE TABLE project_teams (
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    project_id   TEXT NOT NULL,
    team_id      TEXT NOT NULL,
    added_at     TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    removed_at   TEXT,
    PRIMARY KEY (project_id, team_id),
    FOREIGN KEY (project_id, workspace_id) REFERENCES projects (id, workspace_id),
    FOREIGN KEY (team_id, workspace_id) REFERENCES teams (id, workspace_id)
) STRICT;

CREATE INDEX idx_project_teams_team_id ON project_teams (team_id);

CREATE TABLE team_members (
    workspace_id  TEXT NOT NULL REFERENCES workspaces (id),
    membership_id TEXT NOT NULL,
    team_id       TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('lead', 'member')),
    added_at      TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    removed_at    TEXT,
    PRIMARY KEY (membership_id, team_id),
    FOREIGN KEY (membership_id, workspace_id) REFERENCES memberships (id, workspace_id),
    FOREIGN KEY (team_id, workspace_id) REFERENCES teams (id, workspace_id)
) STRICT;

CREATE INDEX idx_team_members_team_id ON team_members (team_id);

CREATE TABLE tasks (
    id           TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    project_id   TEXT NOT NULL,
    status_id    TEXT NOT NULL,
    team_id      TEXT,
    parent_id    TEXT,
    created_by   TEXT NOT NULL,
    title        TEXT NOT NULL,
    description  TEXT,
    position     TEXT NOT NULL,
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    deleted_at   TEXT,
    UNIQUE (id, workspace_id),
    UNIQUE (id, project_id),
    FOREIGN KEY (project_id, workspace_id) REFERENCES projects (id, workspace_id),
    FOREIGN KEY (status_id, project_id) REFERENCES statuses (id, project_id),
    FOREIGN KEY (parent_id, project_id) REFERENCES tasks (id, project_id),
    FOREIGN KEY (created_by, workspace_id) REFERENCES memberships (id, workspace_id),
    FOREIGN KEY (project_id, team_id) REFERENCES project_teams (project_id, team_id)
) STRICT;

CREATE INDEX idx_tasks_parent_id ON tasks (parent_id);
CREATE INDEX idx_tasks_project_id_team_id ON tasks (project_id, team_id);
CREATE INDEX idx_tasks_project_id_status_id_position ON tasks (project_id, status_id, position);
CREATE INDEX idx_tasks_workspace_id_updated_at ON tasks (workspace_id, updated_at);

CREATE TABLE task_assignees (
    workspace_id  TEXT NOT NULL REFERENCES workspaces (id),
    task_id       TEXT NOT NULL,
    membership_id TEXT NOT NULL,
    assigned_at   TEXT NOT NULL,
    updated_at    TEXT NOT NULL,
    removed_at    TEXT,
    PRIMARY KEY (task_id, membership_id),
    FOREIGN KEY (task_id, workspace_id) REFERENCES tasks (id, workspace_id),
    FOREIGN KEY (membership_id, workspace_id) REFERENCES memberships (id, workspace_id)
) STRICT;

CREATE INDEX idx_task_assignees_membership_id
    ON task_assignees (membership_id)
    WHERE removed_at IS NULL;