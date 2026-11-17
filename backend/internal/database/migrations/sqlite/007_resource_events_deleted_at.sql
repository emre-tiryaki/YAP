-- Resetting the local data clears the resource activity log too. Events are
-- soft-deleted like every other synchronized row, so the reset reaches other
-- devices as tombstones; reads skip rows with deleted_at set.
ALTER TABLE resource_events ADD COLUMN deleted_at TEXT;
