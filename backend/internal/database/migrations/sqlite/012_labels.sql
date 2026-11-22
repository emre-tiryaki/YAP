-- Labels belong to a project; a task can carry the labels of its own project.
CREATE TABLE labels (
    id TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    position TEXT NOT NULL,
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT,
    UNIQUE (id, workspace_id), UNIQUE (id, project_id),
    FOREIGN KEY (project_id, workspace_id) REFERENCES projects (id, workspace_id)
) STRICT;
CREATE UNIQUE INDEX idx_labels_project_id_name ON labels (project_id, name) WHERE deleted_at IS NULL;
CREATE INDEX idx_labels_workspace_id_updated_at ON labels (workspace_id, updated_at);

CREATE TABLE task_labels (
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    project_id TEXT NOT NULL,
    task_id TEXT NOT NULL, label_id TEXT NOT NULL,
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL, removed_at TEXT,
    PRIMARY KEY (task_id, label_id),
    FOREIGN KEY (task_id, project_id) REFERENCES tasks (id, project_id),
    FOREIGN KEY (label_id, project_id) REFERENCES labels (id, project_id),
    FOREIGN KEY (project_id, workspace_id) REFERENCES projects (id, workspace_id)
) STRICT;
CREATE INDEX idx_task_labels_label_id ON task_labels (label_id) WHERE removed_at IS NULL;
CREATE INDEX idx_task_labels_workspace_id_updated_at ON task_labels (workspace_id, updated_at);
