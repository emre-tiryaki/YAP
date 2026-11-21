-- Project identifiers (ABC) and per-project task numbers (ABC-12).
ALTER TABLE projects ADD COLUMN identifier TEXT;
ALTER TABLE projects ADD COLUMN next_task_number INTEGER NOT NULL DEFAULT 1;
ALTER TABLE tasks ADD COLUMN number INTEGER;

-- Identifier backfill: the first (up to) 3 ASCII letters of the uppercased
-- name, PRJ when there are fewer than 2. Projects of a workspace that end up
-- with the same base get a number appended in creation order (ABC, ABC2, ...).
WITH RECURSIVE
scan (project_id, pos, letters) AS (
    SELECT p.id, 1,
           CASE WHEN substr(upper(p.name), 1, 1) BETWEEN 'A' AND 'Z' THEN substr(upper(p.name), 1, 1) ELSE '' END
      FROM projects p
    UNION ALL
    SELECT s.project_id, s.pos + 1,
           CASE WHEN length(s.letters) < 3 AND substr(upper(p.name), s.pos + 1, 1) BETWEEN 'A' AND 'Z'
                THEN s.letters || substr(upper(p.name), s.pos + 1, 1)
                ELSE s.letters END
      FROM scan s
      JOIN projects p ON p.id = s.project_id
     WHERE s.pos < length(p.name)
),
final (project_id, base) AS (
    SELECT p.id,
           CASE WHEN length(COALESCE(sc.letters, '')) >= 2 THEN sc.letters ELSE 'PRJ' END
      FROM projects p
      LEFT JOIN scan sc ON sc.project_id = p.id AND sc.pos = (SELECT MAX(pos) FROM scan WHERE project_id = p.id)
),
ranked (project_id, base, rn) AS (
    SELECT f.project_id, f.base,
           ROW_NUMBER() OVER (PARTITION BY p.workspace_id, f.base ORDER BY p.created_at, p.id)
      FROM final f
      JOIN projects p ON p.id = f.project_id
)
UPDATE projects
   SET identifier = (SELECT CASE WHEN r.rn = 1 THEN r.base
                                 ELSE substr(r.base, 1, 5 - length(CAST(r.rn AS TEXT))) || CAST(r.rn AS TEXT) END
                       FROM ranked r WHERE r.project_id = projects.id);

-- Task numbers: 1, 2, ... per project in creation order (deleted tasks keep
-- theirs, so a number is never reused).
UPDATE tasks
   SET number = (SELECT x.rn
                   FROM (SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id ORDER BY created_at, id) AS rn
                           FROM tasks) x
                  WHERE x.id = tasks.id);

UPDATE projects
   SET next_task_number = COALESCE((SELECT MAX(number) FROM tasks WHERE tasks.project_id = projects.id), 0) + 1;

CREATE UNIQUE INDEX idx_projects_workspace_id_identifier
    ON projects (workspace_id, identifier)
    WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_tasks_project_id_number ON tasks (project_id, number);
