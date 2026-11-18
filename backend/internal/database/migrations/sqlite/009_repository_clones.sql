-- The copy of a repository that the app shows is always the ZIP of the
-- newest commit; a git clone is only made into a folder the user picks
-- (repository_clones). Copies that were kept as clones are fetched again as
-- ZIP: their checkouts start over.
UPDATE repository_checkouts
   SET state = 'pending', commit_sha = NULL, fetched_at = NULL, checked_at = NULL, attempted_at = NULL,
       error = NULL, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
 WHERE resource_id IN (SELECT resource_id FROM resource_repositories WHERE method = 'git');

ALTER TABLE resource_repositories DROP COLUMN method;

-- Clones the user made of a repository on this device, so the app can show
-- where they are. Local only, never synchronized: paths belong to one
-- computer. The app never changes a clone after making it; deleted_at only
-- takes it off the list, the files stay.
CREATE TABLE repository_clones (
    id          TEXT NOT NULL PRIMARY KEY CHECK (length(id) = 36),
    resource_id TEXT NOT NULL REFERENCES resource_repositories (resource_id),
    path        TEXT NOT NULL CHECK (length(path) > 0),
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL,
    deleted_at  TEXT
) STRICT;

CREATE INDEX idx_repository_clones_resource_id
    ON repository_clones (resource_id)
    WHERE deleted_at IS NULL;
