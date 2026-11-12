-- A task's finished state no longer follows its status: completed_at is set
-- when the task is marked done and cleared when it is reopened, wherever the
-- task sits on the board. Tasks already in a done-category status keep
-- counting as finished.
ALTER TABLE tasks ADD COLUMN completed_at TEXT;

UPDATE tasks
   SET completed_at = updated_at
 WHERE deleted_at IS NULL
   AND EXISTS (SELECT 1 FROM statuses s
                WHERE s.id = tasks.status_id AND s.plan_id = tasks.plan_id
                  AND s.category = 'done');
