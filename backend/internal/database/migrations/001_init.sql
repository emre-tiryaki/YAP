CREATE TABLE accounts (
	id            INTEGER PRIMARY KEY,
	email         TEXT    NOT NULL UNIQUE COLLATE NOCASE CHECK (length(email) BETWEEN 3 AND 254),
	password_hash TEXT    NOT NULL,
	display_name  TEXT    NOT NULL,
	created_at    INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE teams (
	id         INTEGER PRIMARY KEY,
	name       TEXT    NOT NULL,
	created_at INTEGER NOT NULL DEFAULT (unixepoch())
);

CREATE TABLE memberships (
	id         INTEGER PRIMARY KEY,
	team_id    INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	account_id INTEGER NOT NULL REFERENCES accounts(id),
	role       TEXT    NOT NULL CHECK (role IN ('owner', 'admin', 'member')),
	joined_at  INTEGER NOT NULL DEFAULT (unixepoch()),
	left_at    INTEGER,
	UNIQUE (team_id, account_id),
	UNIQUE (id, team_id)
);
CREATE INDEX idx_memberships_account ON memberships(account_id);

CREATE TABLE projects (
	id          INTEGER PRIMARY KEY,
	team_id     INTEGER NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
	name        TEXT    NOT NULL,
	description TEXT,
	created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
	archived_at INTEGER,
	UNIQUE (team_id, name),
	UNIQUE (id, team_id)
);

CREATE TABLE statuses (
	id          INTEGER PRIMARY KEY,
	project_id  INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	title       TEXT    NOT NULL,
	description TEXT,
	color       TEXT    NOT NULL CHECK (color GLOB '#[0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F][0-9a-fA-F]'),
	category    TEXT    NOT NULL CHECK (category IN ('todo', 'done', 'cancelled')),
	position    INTEGER NOT NULL,
	UNIQUE (project_id, title),
	UNIQUE (id, project_id)
);

CREATE TABLE tasks (
	id          INTEGER PRIMARY KEY,
	team_id     INTEGER NOT NULL,
	project_id  INTEGER NOT NULL,
	status_id   INTEGER NOT NULL,
	parent_id   INTEGER,
	created_by  INTEGER NOT NULL,
	title       TEXT    NOT NULL CHECK (length(title) > 0),
	description TEXT,
	created_at  INTEGER NOT NULL DEFAULT (unixepoch()),
	updated_at  INTEGER NOT NULL DEFAULT (unixepoch()),
	closed_at   INTEGER,
	FOREIGN KEY (project_id, team_id)   REFERENCES projects(id, team_id)    ON DELETE CASCADE,
	FOREIGN KEY (status_id, project_id) REFERENCES statuses(id, project_id),
	FOREIGN KEY (parent_id, project_id) REFERENCES tasks(id, project_id)    ON DELETE CASCADE,
	FOREIGN KEY (created_by, team_id)   REFERENCES memberships(id, team_id),
	CHECK (parent_id IS NULL OR parent_id <> id),
	UNIQUE (id, project_id),
	UNIQUE (id, team_id)
);
CREATE INDEX idx_tasks_project_status ON tasks(project_id, status_id);
CREATE INDEX idx_tasks_parent         ON tasks(parent_id);

CREATE TABLE task_assignees (
	task_id       INTEGER NOT NULL,
	membership_id INTEGER NOT NULL,
	team_id       INTEGER NOT NULL,
	PRIMARY KEY (task_id, membership_id),
	FOREIGN KEY (task_id, team_id)       REFERENCES tasks(id, team_id)       ON DELETE CASCADE,
	FOREIGN KEY (membership_id, team_id) REFERENCES memberships(id, team_id) ON DELETE CASCADE
) WITHOUT ROWID;
CREATE INDEX idx_task_assignees_member ON task_assignees(membership_id);

CREATE TRIGGER trg_tasks_closed_on_insert AFTER INSERT ON tasks
BEGIN
	UPDATE tasks
	SET closed_at = unixepoch()
	WHERE id = NEW.id
	  AND (SELECT category FROM statuses WHERE id = NEW.status_id) IN ('done', 'cancelled');
END;

CREATE TRIGGER trg_tasks_closed_on_status AFTER UPDATE OF status_id ON tasks
BEGIN
	UPDATE tasks
	SET closed_at = CASE
			WHEN (SELECT category FROM statuses WHERE id = NEW.status_id) IN ('done', 'cancelled')
			THEN unixepoch()
		END,
		updated_at = unixepoch()
	WHERE id = NEW.id;
END;