-- A task's priority. 'none' is the default, so existing tasks keep working.
ALTER TABLE tasks ADD COLUMN priority TEXT NOT NULL DEFAULT 'none'
    CHECK (priority IN ('urgent', 'high', 'medium', 'low', 'none'));
