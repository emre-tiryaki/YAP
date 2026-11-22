package service

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	// ExportFormat and ExportVersion identify the layout of data.json.
	ExportFormat  = "yap-export"
	ExportVersion = 1

	exportDataName = "data.json"
	exportFilesDir = "files/"
)

type AccountRepository interface {
	Export(ctx context.Context, q database.DBTX, userID, workspaceID string) ([]model.ExportTable, error)
	Reset(ctx context.Context, q database.DBTX, workspaceID, now string) error

	RowState(ctx context.Context, q database.DBTX, workspaceID, table, id string) (model.ImportRowState, error)
	HasContent(ctx context.Context, q database.DBTX, workspaceID string) (bool, error)
	PlanNames(ctx context.Context, q database.DBTX, workspaceID, projectID string) (map[string]bool, error)
	InsertProject(ctx context.Context, q database.DBTX, workspaceID string, p model.ImportProject, now string) error
	LiveIdentifiers(ctx context.Context, q database.DBTX, workspaceID string) (map[string]bool, error)
	TaskNumber(ctx context.Context, q database.DBTX, taskID string) (int, error)
	ReserveTaskNumber(ctx context.Context, q database.DBTX, workspaceID, projectID, now string) (int, error)
	InsertPlan(ctx context.Context, q database.DBTX, workspaceID string, p model.ImportPlan, now string) error
	InsertStatus(ctx context.Context, q database.DBTX, workspaceID string, s model.ImportStatus, now string) error
	LabelIDByName(ctx context.Context, q database.DBTX, workspaceID, projectID, name string) (string, bool, error)
	InsertLabel(ctx context.Context, q database.DBTX, workspaceID string, l model.ImportLabel, now string) error
	InsertTaskLabel(ctx context.Context, q database.DBTX, workspaceID, projectID string, l model.ImportTaskLabel, now string) error
	InsertTask(ctx context.Context, q database.DBTX, workspaceID, createdBy string, t model.ImportTask, number int, now string) error
	InsertAssignee(ctx context.Context, q database.DBTX, workspaceID, taskID, membershipID, assignedAt, now string) error
	InsertComment(ctx context.Context, q database.DBTX, workspaceID, authorID string, c model.ImportComment) error
	InsertResource(ctx context.Context, q database.DBTX, workspaceID, createdBy, nameKey string, r model.ImportResource, now string) error
	InsertTaskResource(ctx context.Context, q database.DBTX, workspaceID, linkedBy string, l model.ImportTaskResource, now string) error
	InsertEvent(ctx context.Context, q database.DBTX, workspaceID, actorID string, e model.ImportEvent) error
	InsertTaskEvent(ctx context.Context, q database.DBTX, workspaceID, actorID string, e model.ImportTaskEvent) error
	InsertTaskRelation(ctx context.Context, q database.DBTX, workspaceID string, r model.ImportTaskRelation, now string) error
	InsertRepository(ctx context.Context, q database.DBTX, workspaceID, addedBy string, r model.ImportRepository, now string) error
	InsertAIConnection(ctx context.Context, q database.DBTX, workspaceID, userID string, c model.ImportAIConnection) error
	InsertAIChat(ctx context.Context, q database.DBTX, workspaceID, userID string, c model.ImportAIChat) error
	InsertAIMessage(ctx context.Context, q database.DBTX, workspaceID string, m model.ImportAIMessage) error
	InsertAIAttachment(ctx context.Context, q database.DBTX, workspaceID string, a model.ImportAIAttachment) error
}

// AIResetRepository deletes the AI data of a user.
type AIResetRepository interface {
	SoftDeleteAllChats(ctx context.Context, q database.DBTX, workspaceID, userID, now string) error
	SoftDeleteAllConnections(ctx context.Context, q database.DBTX, workspaceID, userID, now string) ([]string, error)
}

// AIStopper ends the answer being written, if any.
type AIStopper interface {
	StopAll()
}

// ResetOptions choose what a reset deletes besides the workspace content.
type ResetOptions struct {
	// AIChats deletes the assistant's chats.
	AIChats bool
	// AIConnections deletes the assistant's connections and their keys.
	AIConnections bool
}

// AccountResources is the part of the resource repository an export needs.
type AccountResources interface {
	ListLive(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Resource, error)
}

// AccountStore is the part of the resource store an export, an import and a
// reset need.
type AccountStore interface {
	Stage(ctx context.Context, r io.Reader) (model.StagedFile, error)
	Discard(staged model.StagedFile)
	Place(staged model.StagedFile, path []string) error
	MakeDir(path []string) error
	Open(path []string) (*os.File, error)
	Remove(path []string) error
}

// AccountService exports, imports and resets everything the local user has.
type AccountService struct {
	db         *sql.DB
	repo       AccountRepository
	resources  AccountResources
	workspaces WorkspaceRepository
	settings   SettingsRepository
	store      AccountStore
	// defaultDisplayName is the name a reset gives the user back, the same
	// one the first start gave them.
	defaultDisplayName string
	now                func() time.Time

	// The AI parts are optional; see WithAI.
	ai      AIResetRepository
	aiKeys  AIKeyStore
	aiStops AIStopper
}

func NewAccountService(db *sql.DB, repo AccountRepository, resources AccountResources, workspaces WorkspaceRepository,
	settings SettingsRepository, store AccountStore, defaultDisplayName string) *AccountService {
	return &AccountService{
		db: db, repo: repo, resources: resources, workspaces: workspaces, settings: settings, store: store,
		defaultDisplayName: defaultDisplayName, now: time.Now,
	}
}

// WithAI lets a reset delete the assistant's chats and connections (and
// their keys) and stop the answer being written.
func (s *AccountService) WithAI(repo AIResetRepository, keys AIKeyStore, stopper AIStopper) *AccountService {
	s.ai, s.aiKeys, s.aiStops = repo, keys, stopper
	return s
}

// exportDocument is the content of data.json.
type exportDocument struct {
	Format     string                      `json:"format"`
	Version    int                         `json:"version"`
	ExportedAt string                      `json:"exported_at"`
	Tables     map[string][]map[string]any `json:"tables"`
	// MissingFiles lists files whose bytes were gone from the disk.
	MissingFiles []string `json:"missing_files"`
}

type exportEntry struct {
	path []string
	r    model.Resource
}

// Export writes a ZIP archive to w: data.json with the live rows of the
// workspace, and the bytes of the live (not trashed) resources under files/,
// laid out like the resource tree. Nothing is written to w when reading the
// database fails.
func (s *AccountService) Export(ctx context.Context, session model.Session, w io.Writer) error {
	var (
		tables    []model.ExportTable
		resources []model.Resource
	)
	// One transaction, so the rows form a consistent snapshot.
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		if tables, err = s.repo.Export(ctx, tx, session.UserID, session.WorkspaceID); err != nil {
			return err
		}
		resources, err = s.resources.ListLive(ctx, tx, session.WorkspaceID)
		return err
	})
	if err != nil {
		return err
	}

	doc := exportDocument{
		Format:       ExportFormat,
		Version:      ExportVersion,
		ExportedAt:   s.now().UTC().Format(time.RFC3339),
		Tables:       make(map[string][]map[string]any, len(tables)),
		MissingFiles: []string{},
	}
	for _, t := range tables {
		doc.Tables[t.Name] = t.Rows
	}

	zw := zip.NewWriter(w)
	for _, e := range treePaths(resources) {
		name := exportFilesDir + strings.Join(e.path, "/")
		if e.r.Kind == model.ResourceFolder {
			if _, err := zw.CreateHeader(&zip.FileHeader{Name: name + "/", Modified: e.r.UpdatedAt}); err != nil {
				return fmt.Errorf("export folder: %w", err)
			}
			continue
		}
		missing, err := s.exportFile(ctx, zw, session.WorkspaceID, name, e)
		if err != nil {
			return err
		}
		if missing {
			doc.MissingFiles = append(doc.MissingFiles, strings.Join(e.path, "/"))
		}
	}

	dw, err := zw.CreateHeader(&zip.FileHeader{Name: exportDataName, Method: zip.Deflate, Modified: s.now()})
	if err != nil {
		return fmt.Errorf("export data: %w", err)
	}
	enc := json.NewEncoder(dw)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("export data: %w", err)
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("export close: %w", err)
	}
	return nil
}

// exportFile copies one file's bytes into the archive and reports whether
// they were missing from the disk.
func (s *AccountService) exportFile(ctx context.Context, zw *zip.Writer, workspaceID, name string, e exportEntry) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	f, err := s.store.Open(append([]string{workspaceID}, e.path...))
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()
	fw, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Deflate, Modified: e.r.UpdatedAt})
	if err != nil {
		return false, fmt.Errorf("export file: %w", err)
	}
	if _, err := io.Copy(fw, f); err != nil {
		return false, fmt.Errorf("export file: %w", err)
	}
	return false, nil
}

// treePaths gives every live resource its path from the top of the tree,
// sorted so folders come before what they contain.
func treePaths(resources []model.Resource) []exportEntry {
	byID := make(map[string]model.Resource, len(resources))
	for _, r := range resources {
		byID[r.ID] = r
	}
	entries := make([]exportEntry, 0, len(resources))
	for _, r := range resources {
		path := []string{r.Name}
		complete := true
		for parent := r.ParentID; parent != ""; {
			p, ok := byID[parent]
			if !ok {
				complete = false // not reachable from the top; cannot happen for live rows
				break
			}
			path = append([]string{p.Name}, path...)
			parent = p.ParentID
		}
		if complete {
			entries = append(entries, exportEntry{path: path, r: r})
		}
	}
	slices.SortFunc(entries, func(a, b exportEntry) int { return slices.Compare(a.path, b.path) })
	return entries
}

// Reset deletes everything the user has in the workspace and puts their
// settings and names back to the defaults of a first start. The ids of the
// user, the workspace and the owner membership are kept. Rows are
// soft-deleted in one transaction; the resource bytes, trash included, are
// removed from the disk afterwards. A failure there is returned, and running
// the reset again finishes it. AI chats and connections are only deleted
// when opts say so; a running answer is stopped first.
func (s *AccountService) Reset(ctx context.Context, session model.Session, opts ResetOptions) error {
	if s.aiStops != nil {
		s.aiStops.StopAll()
	}
	var removedKeys []string
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		removedKeys = nil
		now := s.now().UTC().Format(time.RFC3339)
		if err := s.repo.Reset(ctx, tx, session.WorkspaceID, now); err != nil {
			return err
		}
		if s.ai != nil && opts.AIChats {
			if err := s.ai.SoftDeleteAllChats(ctx, tx, session.WorkspaceID, session.UserID, now); err != nil {
				return err
			}
		}
		if s.ai != nil && opts.AIConnections {
			ids, err := s.ai.SoftDeleteAllConnections(ctx, tx, session.WorkspaceID, session.UserID, now)
			if err != nil {
				return err
			}
			removedKeys = ids
		}
		if err := resetUserSettings(ctx, tx, s.settings, session.UserID, now); err != nil {
			return err
		}
		if err := s.workspaces.UpdateDisplayName(ctx, tx, session.UserID, s.defaultDisplayName, now); err != nil {
			return err
		}
		return s.workspaces.UpdateName(ctx, tx, session.WorkspaceID, personalWorkspaceName, now)
	})
	if err != nil {
		return err
	}
	for _, id := range removedKeys {
		if s.aiKeys == nil {
			break
		}
		if err := s.aiKeys.Delete(id); err != nil {
			slog.Error("delete ai key", "error", err)
		}
	}
	if err := s.store.Remove([]string{session.WorkspaceID}); err != nil {
		return fmt.Errorf("reset resource files: %w", err)
	}
	return nil
}
