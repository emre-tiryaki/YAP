package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// RepoRepository keeps the GitHub repositories of resource folders and the
// state of this device's copies.
type RepoRepository struct{}

func NewRepoRepository() *RepoRepository {
	return &RepoRepository{}
}

const repoColumns = `rr.resource_id, rr.github_id, rr.full_name, rr.html_url, rr.branch, rr.is_private,
	rr.added_by, rr.created_at, rr.updated_at, r.trashed_at,
	COALESCE(rc.state, 'pending'), rc.commit_sha, rc.fetched_at, rc.checked_at, rc.error, rc.attempted_at`

const repoFrom = `FROM resource_repositories rr
	JOIN resources r ON r.id = rr.resource_id AND r.workspace_id = rr.workspace_id
	LEFT JOIN repository_checkouts rc ON rc.resource_id = rr.resource_id`

func parseOptionalTime(v sql.NullString, field string) (*time.Time, error) {
	if !v.Valid {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v.String)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", field, err)
	}
	return &t, nil
}

func scanRepo(row rowScanner, workspaceID string) (model.Repository, error) {
	r := model.Repository{WorkspaceID: workspaceID}
	var createdAt, updatedAt, state string
	var isPrivate int
	var trashedAt, commitSHA, fetchedAt, checkedAt, checkoutErr, attemptedAt sql.NullString
	if err := row.Scan(&r.ResourceID, &r.GitHubID, &r.FullName, &r.HTMLURL, &r.Branch, &isPrivate,
		&r.AddedBy, &createdAt, &updatedAt, &trashedAt,
		&state, &commitSHA, &fetchedAt, &checkedAt, &checkoutErr, &attemptedAt); err != nil {
		return model.Repository{}, err
	}
	r.Private = isPrivate == 1
	r.State = model.CheckoutState(state)
	r.CommitSHA = commitSHA.String
	r.Error = model.CheckoutError(checkoutErr.String)
	var err error
	if r.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
		return model.Repository{}, fmt.Errorf("parse created_at: %w", err)
	}
	if r.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt); err != nil {
		return model.Repository{}, fmt.Errorf("parse updated_at: %w", err)
	}
	if r.AttemptedAt, err = parseOptionalTime(attemptedAt, "attempted_at"); err != nil {
		return model.Repository{}, err
	}
	if r.TrashedAt, err = parseOptionalTime(trashedAt, "trashed_at"); err != nil {
		return model.Repository{}, err
	}
	if r.FetchedAt, err = parseOptionalTime(fetchedAt, "fetched_at"); err != nil {
		return model.Repository{}, err
	}
	if r.CheckedAt, err = parseOptionalTime(checkedAt, "checked_at"); err != nil {
		return model.Repository{}, err
	}
	return r, nil
}

// Insert adds the repository of a folder with a pending copy. A row left
// behind by a purged folder with the same id (brought back by an import) is
// overwritten.
func (*RepoRepository) Insert(ctx context.Context, q database.DBTX, r model.Repository, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO resource_repositories (resource_id, workspace_id, github_id, full_name, html_url, branch,
		                                    is_private, added_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (resource_id) DO UPDATE SET
		     github_id = excluded.github_id, full_name = excluded.full_name, html_url = excluded.html_url,
		     branch = excluded.branch, is_private = excluded.is_private,
		     added_by = excluded.added_by, created_at = excluded.created_at, updated_at = excluded.updated_at`,
		r.ResourceID, r.WorkspaceID, r.GitHubID, r.FullName, r.HTMLURL, r.Branch,
		boolInt(r.Private), r.AddedBy, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert repository: %w", err)
	}
	_, err = q.ExecContext(ctx,
		`INSERT INTO repository_checkouts (resource_id, state, updated_at) VALUES (?, 'pending', ?)
		 ON CONFLICT (resource_id) DO UPDATE SET
		     state = 'pending', commit_sha = NULL, fetched_at = NULL, checked_at = NULL, attempted_at = NULL,
		     error = NULL, updated_at = excluded.updated_at`,
		r.ResourceID, now,
	)
	if err != nil {
		return fmt.Errorf("insert checkout: %w", err)
	}
	return nil
}

// Get returns the repository of a folder that has not been purged.
func (*RepoRepository) Get(ctx context.Context, q database.DBTX, workspaceID, resourceID string) (model.Repository, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+repoColumns+` `+repoFrom+`
		 WHERE rr.resource_id = ? AND rr.workspace_id = ? AND r.deleted_at IS NULL`,
		resourceID, workspaceID,
	)
	r, err := scanRepo(row, workspaceID)
	if err != nil {
		return model.Repository{}, fmt.Errorf("get repository: %w", err)
	}
	return r, nil
}

// List returns the repositories of the workspace whose folders have not been
// purged, trashed ones included.
func (*RepoRepository) List(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Repository, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT `+repoColumns+` `+repoFrom+`
		 WHERE rr.workspace_id = ? AND r.deleted_at IS NULL
		 ORDER BY rr.resource_id`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	defer rows.Close()
	out := []model.Repository{}
	for rows.Next() {
		r, err := scanRepo(rows, workspaceID)
		if err != nil {
			return nil, fmt.Errorf("scan repository: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateInfo stores what GitHub says about a repository now (it may have been
// renamed, moved or made private).
func (*RepoRepository) UpdateInfo(ctx context.Context, q database.DBTX, workspaceID, resourceID, fullName, htmlURL string, private bool, now string) error {
	_, err := q.ExecContext(ctx,
		`UPDATE resource_repositories SET full_name = ?, html_url = ?, is_private = ?, updated_at = ?
		 WHERE resource_id = ? AND workspace_id = ?`,
		fullName, htmlURL, boolInt(private), now, resourceID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("update repository: %w", err)
	}
	return nil
}

// SaveCheckout writes the state of this device's copy of a repository.
func (*RepoRepository) SaveCheckout(ctx context.Context, q database.DBTX, r model.Repository, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO repository_checkouts (resource_id, state, commit_sha, fetched_at, checked_at, attempted_at,
		                                   error, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (resource_id) DO UPDATE SET
		     state = excluded.state, commit_sha = excluded.commit_sha, fetched_at = excluded.fetched_at,
		     checked_at = excluded.checked_at, attempted_at = excluded.attempted_at, error = excluded.error,
		     updated_at = excluded.updated_at`,
		r.ResourceID, string(r.State), nullIfEmpty(r.CommitSHA), formatOptionalTime(r.FetchedAt),
		formatOptionalTime(r.CheckedAt), formatOptionalTime(r.AttemptedAt), nullIfEmpty(string(r.Error)), now,
	)
	if err != nil {
		return fmt.Errorf("save checkout: %w", err)
	}
	return nil
}

func formatOptionalTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

// InsertClone records a clone the user made.
func (*RepoRepository) InsertClone(ctx context.Context, q database.DBTX, c model.RepositoryClone, now string) error {
	_, err := q.ExecContext(ctx,
		`INSERT INTO repository_clones (id, resource_id, path, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.ResourceID, c.Path, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert clone: %w", err)
	}
	return nil
}

// ListClones returns the clones of a repository still on the list, newest
// first.
func (*RepoRepository) ListClones(ctx context.Context, q database.DBTX, resourceID string) ([]model.RepositoryClone, error) {
	rows, err := q.QueryContext(ctx,
		`SELECT id, path, created_at FROM repository_clones
		 WHERE resource_id = ? AND deleted_at IS NULL
		 ORDER BY id DESC`,
		resourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list clones: %w", err)
	}
	defer rows.Close()
	out := []model.RepositoryClone{}
	for rows.Next() {
		c := model.RepositoryClone{ResourceID: resourceID}
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Path, &createdAt); err != nil {
			return nil, fmt.Errorf("scan clone: %w", err)
		}
		if c.CreatedAt, err = time.Parse(time.RFC3339, createdAt); err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ForgetClone takes a clone off the list; it reports whether it was there.
func (*RepoRepository) ForgetClone(ctx context.Context, q database.DBTX, resourceID, id, now string) (bool, error) {
	res, err := q.ExecContext(ctx,
		`UPDATE repository_clones SET deleted_at = ?, updated_at = ?
		 WHERE id = ? AND resource_id = ? AND deleted_at IS NULL`,
		now, now, id, resourceID,
	)
	if err != nil {
		return false, fmt.Errorf("forget clone: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}
