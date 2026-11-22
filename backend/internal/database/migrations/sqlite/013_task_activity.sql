-- Start date, the activity log of a task and the relations between tasks.
ALTER TABLE tasks ADD COLUMN start_date TEXT;

-- One row per changed field of a task. details is a JSON object of strings.
CREATE TABLE task_events (
    id TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    task_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    action TEXT NOT NULL CHECK (action IN (
        'created', 'renamed', 'described', 'status_changed', 'priority_changed',
        'due_changed', 'start_changed', 'labels_changed', 'done', 'reopened',
        'relation_added', 'relation_removed')),
    details TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT,
    FOREIGN KEY (task_id, workspace_id) REFERENCES tasks (id, workspace_id),
    FOREIGN KEY (actor_id, workspace_id) REFERENCES memberships (id, workspace_id)
) STRICT;
CREATE INDEX idx_task_events_task_id_id ON task_events (task_id, id);
CREATE INDEX idx_task_events_workspace_id_updated_at ON task_events (workspace_id, updated_at);

-- task_id blocks / relates to / duplicates related_task_id. Tasks of
-- different projects can be related, so the keys are workspace wide.
CREATE TABLE task_relations (
    workspace_id TEXT NOT NULL REFERENCES workspaces (id),
    project_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    related_task_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('blocks', 'relates', 'duplicates')),
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL, removed_at TEXT,
    PRIMARY KEY (task_id, related_task_id),
    CHECK (task_id <> related_task_id),
    FOREIGN KEY (task_id, workspace_id) REFERENCES tasks (id, workspace_id),
    FOREIGN KEY (related_task_id, workspace_id) REFERENCES tasks (id, workspace_id)
) STRICT;
CREATE INDEX idx_task_relations_related_task_id ON task_relations (related_task_id) WHERE removed_at IS NULL;
CREATE INDEX idx_task_relations_workspace_id_updated_at ON task_relations (workspace_id, updated_at);
