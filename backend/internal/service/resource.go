package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	// MaxUploadBytes is the largest file that can be uploaded.
	MaxUploadBytes = 100 << 20
	// MaxTextBytes is the largest file that counts as editable text.
	MaxTextBytes = 20 << 20
	// trashDirName holds trashed items inside a workspace's directory; the
	// name is reserved at the top of the tree.
	trashDirName = ".yap-trash"
	// editMergeWindow folds repeated saves by the same member into one
	// 'edited' event while they keep coming within this window.
	editMergeWindow  = 10 * time.Minute
	maxResourceName  = 255
	defaultEventPage = 50
	maxEventPage     = 100
)

// OnConflict says what to do when an item with the same name already exists
// in the target folder.
type OnConflict string

const (
	ConflictFail OnConflict = "fail"
	// ConflictKeepBoth picks a free name such as "notes (1).txt".
	ConflictKeepBoth OnConflict = "keep_both"
	// ConflictReplace overwrites (uploads) or trashes (move, copy) the
	// existing file. It is not allowed when either side is a folder.
	ConflictReplace OnConflict = "replace"
)

func (c OnConflict) valid() bool {
	return c == ConflictFail || c == ConflictKeepBoth || c == ConflictReplace
}

type ResourceRepository interface {
	Insert(ctx context.Context, q database.DBTX, r model.Resource, nameKey, now string) error
	Get(ctx context.Context, q database.DBTX, workspaceID, id string) (model.Resource, error)
	ListLive(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Resource, error)
	ListTrash(ctx context.Context, q database.DBTX, workspaceID string) ([]model.Resource, error)
	Ancestors(ctx context.Context, q database.DBTX, workspaceID, id string) ([]model.Resource, error)
	LiveSubtree(ctx context.Context, q database.DBTX, workspaceID, id string) ([]model.Resource, error)
	SiblingNameKeys(ctx context.Context, q database.DBTX, workspaceID, parentID string) (map[string]string, error)
	Place(ctx context.Context, q database.DBTX, workspaceID, id, parentID, name, nameKey, now string) error
	UpdateContent(ctx context.Context, q database.DBTX, workspaceID, id, mimeType string, size int64, isText bool, now string) error
	Trash(ctx context.Context, q database.DBTX, workspaceID, id, now string) error
	Restore(ctx context.Context, q database.DBTX, workspaceID, rootID, now string) error
	Purge(ctx context.Context, q database.DBTX, workspaceID, rootID, now string) error
	CloseLinksOfTrashGroup(ctx context.Context, q database.DBTX, workspaceID, rootID, now string) error
	Link(ctx context.Context, q database.DBTX, workspaceID, taskID, resourceID, linkedBy, now string) (bool, error)
	Unlink(ctx context.Context, q database.DBTX, workspaceID, taskID, resourceID, now string) (bool, error)
	ListByTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) ([]model.Resource, error)
	ListTasks(ctx context.Context, q database.DBTX, workspaceID, resourceID string) ([]model.LinkedTask, error)
	InsertEvent(ctx context.Context, q database.DBTX, e model.ResourceEvent, now string) error
	LastEvent(ctx context.Context, q database.DBTX, workspaceID, resourceID string) (model.ResourceEvent, error)
	ExtendEvent(ctx context.Context, q database.DBTX, workspaceID, eventID, details, now string) error
	ListEvents(ctx context.Context, q database.DBTX, workspaceID string, scope model.ResourceEventScope, before string, limit int) ([]model.ResourceEvent, error)
}

// ResourceStore keeps the bytes. Paths are segments under the store's root.
type ResourceStore interface {
	Stage(ctx context.Context, r io.Reader) (model.StagedFile, error)
	Discard(staged model.StagedFile)
	Place(staged model.StagedFile, path []string) error
	Open(path []string) (*os.File, error)
	MakeDir(path []string) error
	Move(from, to []string) error
	Copy(from, to []string) error
	Remove(path []string) error
}

// ResourceTaskLookup is the part of the task repository links need.
type ResourceTaskLookup interface {
	Get(ctx context.Context, q database.DBTX, workspaceID, taskID string) (model.Task, error)
}

type ResourceService struct {
	db    *sql.DB
	repo  ResourceRepository
	tasks ResourceTaskLookup
	store ResourceStore
	// now is replaceable so tests can step over the edit merge window.
	now func() time.Time
}

func NewResourceService(db *sql.DB, repo ResourceRepository, tasks ResourceTaskLookup, store ResourceStore) *ResourceService {
	return &ResourceService{db: db, repo: repo, tasks: tasks, store: store, now: time.Now}
}

func (s *ResourceService) nowString() string {
	return s.now().UTC().Format(time.RFC3339)
}

// normalizeResourceName trims the name and rejects what common file systems
// do not allow.
func normalizeResourceName(name string) (string, error) {
	name = strings.TrimSpace(name)
	switch {
	case name == "", name == ".", name == "..":
		return "", fmt.Errorf("%w: name is required", ErrInvalidInput)
	case len(name) > maxResourceName:
		return "", fmt.Errorf("%w: name is too long", ErrInvalidInput)
	case !utf8.ValidString(name):
		return "", fmt.Errorf("%w: name is not valid UTF-8", ErrInvalidInput)
	case strings.ContainsAny(name, `/\:*?"<>|`):
		return "", fmt.Errorf("%w: name contains a forbidden character", ErrInvalidInput)
	case strings.IndexFunc(name, unicode.IsControl) >= 0:
		return "", fmt.Errorf("%w: name contains a control character", ErrInvalidInput)
	}
	return name, nil
}

// nameKey is what sibling names are compared by: case-insensitively.
func nameKey(name string) string {
	return strings.ToLower(name)
}

func checkReservedName(parentID, name string) error {
	if parentID == "" && nameKey(name) == trashDirName {
		return fmt.Errorf("%w: name is reserved", ErrInvalidInput)
	}
	return nil
}

// freeName returns name, or the first "name (n).ext" not in taken.
func freeName(name string, kind model.ResourceKind, taken map[string]string) string {
	if _, ok := taken[nameKey(name)]; !ok {
		return name
	}
	base, ext := name, ""
	if kind == model.ResourceFile {
		if e := filepath.Ext(name); e != "" && e != name {
			base, ext = strings.TrimSuffix(name, e), e
		}
	}
	for n := 1; ; n++ {
		suffix := " (" + strconv.Itoa(n) + ")"
		b := base
		if len(b)+len(suffix)+len(ext) > maxResourceName {
			b = truncateUTF8(b, maxResourceName-len(suffix)-len(ext))
		}
		candidate := b + suffix + ext
		if _, ok := taken[nameKey(candidate)]; !ok {
			return candidate
		}
	}
}

func truncateUTF8(s string, n int) string {
	if n <= 0 {
		return ""
	}
	for len(s) > n {
		_, size := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-size]
	}
	return s
}

// detectMime picks a content type from the extension, then the client's
// header, then the bytes.
func detectMime(name, header string, head []byte) string {
	if t := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); t != "" {
		return baseMime(t)
	}
	if t := baseMime(header); t != "" && t != "application/octet-stream" {
		return t
	}
	return baseMime(http.DetectContentType(head))
}

func baseMime(t string) string {
	if t == "" {
		return ""
	}
	mt, _, err := mime.ParseMediaType(t)
	if err != nil {
		return ""
	}
	return mt
}

func details(fields map[string]any) string {
	b, err := json.Marshal(fields)
	if err != nil {
		// Only strings and numbers go in; this cannot fail.
		return "{}"
	}
	return string(b)
}

func notFound(err error, what string) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%w: %s", ErrNotFound, what)
	}
	return err
}

// tolerateMissing ignores bytes that vanished from the disk behind the app's
// back: the database stays the source of truth for the tree.
func tolerateMissing(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// live returns a resource that is neither trashed nor purged.
func (s *ResourceService) live(ctx context.Context, q database.DBTX, workspaceID, id string) (model.Resource, error) {
	r, err := s.repo.Get(ctx, q, workspaceID, id)
	if err != nil {
		return model.Resource{}, notFound(err, "resource")
	}
	if r.TrashedAt != nil {
		return model.Resource{}, fmt.Errorf("%w: resource is in the trash", ErrNotFound)
	}
	return r, nil
}

// requireFolder checks that parentID is "" (the top) or a live folder.
func (s *ResourceService) requireFolder(ctx context.Context, q database.DBTX, workspaceID, parentID string) error {
	if parentID == "" {
		return nil
	}
	r, err := s.live(ctx, q, workspaceID, parentID)
	if err != nil {
		return err
	}
	if r.Kind != model.ResourceFolder {
		return fmt.Errorf("%w: parent is not a folder", ErrInvalidInput)
	}
	return nil
}

// diskPath is where the bytes of a resource live: under the workspace
// directory while live, under the trash directory of its trash root while
// trashed.
func (s *ResourceService) diskPath(ctx context.Context, q database.DBTX, workspaceID, id string) ([]string, error) {
	chain, err := s.repo.Ancestors(ctx, q, workspaceID, id)
	if err != nil {
		return nil, err
	}
	if len(chain) == 0 {
		return nil, fmt.Errorf("%w: resource", ErrNotFound)
	}
	rootID := chain[0].TrashedRootID
	var names []string
	for _, r := range chain {
		names = append([]string{r.Name}, names...)
		if rootID != "" && r.ID == rootID {
			break
		}
	}
	if rootID != "" {
		return append([]string{workspaceID, trashDirName, rootID}, names...), nil
	}
	return append([]string{workspaceID}, names...), nil
}

// displayPath is the "/"-joined location of a live folder, "" for the top.
func (s *ResourceService) displayPath(ctx context.Context, q database.DBTX, workspaceID, folderID string) (string, error) {
	if folderID == "" {
		return "", nil
	}
	chain, err := s.repo.Ancestors(ctx, q, workspaceID, folderID)
	if err != nil {
		return "", err
	}
	names := make([]string, len(chain))
	for i, r := range chain {
		names[len(chain)-1-i] = r.Name
	}
	return strings.Join(names, "/"), nil
}

func (s *ResourceService) logEvent(ctx context.Context, q database.DBTX, session model.Session, resourceID, taskID string, action model.ResourceAction, fields map[string]any, now string) error {
	return s.repo.InsertEvent(ctx, q, model.ResourceEvent{
		ID:          model.NewID(),
		WorkspaceID: session.WorkspaceID,
		ResourceID:  resourceID,
		ActorID:     session.MembershipID,
		Action:      action,
		TaskID:      taskID,
		Details:     details(fields),
	}, now)
}

// List returns every live resource of the workspace; the client builds the
// tree from the parent ids.
func (s *ResourceService) List(ctx context.Context, session model.Session) ([]model.Resource, error) {
	return s.repo.ListLive(ctx, s.db, session.WorkspaceID)
}

func (s *ResourceService) Trash(ctx context.Context, session model.Session) ([]model.Resource, error) {
	return s.repo.ListTrash(ctx, s.db, session.WorkspaceID)
}

func (s *ResourceService) Get(ctx context.Context, session model.Session, id string) (model.Resource, error) {
	r, err := s.repo.Get(ctx, s.db, session.WorkspaceID, id)
	if err != nil {
		return model.Resource{}, notFound(err, "resource")
	}
	return r, nil
}

// resolveName validates name for a new item in parentID and applies the
// conflict policy. It returns the name to use and, for replace, the id of the
// existing item that has the name.
func (s *ResourceService) resolveName(ctx context.Context, q database.DBTX, workspaceID, parentID, name string, kind model.ResourceKind, onConflict OnConflict, selfID string) (string, string, error) {
	name, err := normalizeResourceName(name)
	if err != nil {
		return "", "", err
	}
	if err := checkReservedName(parentID, name); err != nil {
		return "", "", err
	}
	taken, err := s.repo.SiblingNameKeys(ctx, q, workspaceID, parentID)
	if err != nil {
		return "", "", err
	}
	if selfID != "" && taken[nameKey(name)] == selfID {
		delete(taken, nameKey(name))
	}
	existingID, exists := taken[nameKey(name)]
	if !exists {
		return name, "", nil
	}
	switch onConflict {
	case ConflictKeepBoth:
		return freeName(name, kind, taken), "", nil
	case ConflictReplace:
		existing, err := s.repo.Get(ctx, q, workspaceID, existingID)
		if err != nil {
			return "", "", err
		}
		if existing.Kind != model.ResourceFile || kind != model.ResourceFile {
			return "", "", fmt.Errorf("%w: only a file can replace a file", ErrConflict)
		}
		return name, existingID, nil
	default:
		return "", "", fmt.Errorf("%w: name is taken", ErrConflict)
	}
}

func (s *ResourceService) CreateFolder(ctx context.Context, session model.Session, parentID, name string) (model.Resource, error) {
	var created model.Resource
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requireFolder(ctx, tx, session.WorkspaceID, parentID); err != nil {
			return err
		}
		name, _, err := s.resolveName(ctx, tx, session.WorkspaceID, parentID, name, model.ResourceFolder, ConflictFail, "")
		if err != nil {
			return err
		}
		now := s.nowString()
		r := model.Resource{
			ID: model.NewID(), WorkspaceID: session.WorkspaceID, ParentID: parentID,
			Kind: model.ResourceFolder, Name: name, CreatedBy: session.MembershipID,
		}
		if err := s.repo.Insert(ctx, tx, r, nameKey(name), now); err != nil {
			return err
		}
		if err := s.logEvent(ctx, tx, session, r.ID, "", model.ResourceCreated, map[string]any{"name": name}, now); err != nil {
			return err
		}
		path, err := s.diskPath(ctx, tx, session.WorkspaceID, r.ID)
		if err != nil {
			return err
		}
		if err := s.store.MakeDir(path); err != nil {
			return err
		}
		created, err = s.repo.Get(ctx, tx, session.WorkspaceID, r.ID)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return created, nil
}

// CreateFile creates an empty file, ready to be edited.
func (s *ResourceService) CreateFile(ctx context.Context, session model.Session, parentID, name string) (model.Resource, error) {
	return s.put(ctx, session, UploadInput{ParentID: parentID, Name: name, OnConflict: ConflictFail}, strings.NewReader(""), model.ResourceCreated)
}

type UploadInput struct {
	ParentID   string
	Name       string
	OnConflict OnConflict
	// ContentType is the type the client sent, used when the name and bytes
	// do not tell.
	ContentType string
}

// Upload stores a file in a folder, applying the conflict policy when the
// name is taken. Replacing keeps the existing item (and its task links) and
// gives it the new content.
func (s *ResourceService) Upload(ctx context.Context, session model.Session, in UploadInput, body io.Reader) (model.Resource, error) {
	if !in.OnConflict.valid() {
		return model.Resource{}, fmt.Errorf("%w: on_conflict", ErrInvalidInput)
	}
	return s.put(ctx, session, in, body, model.ResourceUploaded)
}

func (s *ResourceService) stage(ctx context.Context, body io.Reader, limit int64) (model.StagedFile, error) {
	staged, err := s.store.Stage(ctx, io.LimitReader(body, limit+1))
	if err != nil {
		return model.StagedFile{}, err
	}
	if staged.Size > limit {
		s.store.Discard(staged)
		return model.StagedFile{}, fmt.Errorf("%w: body", ErrTooLarge)
	}
	return staged, nil
}

func (s *ResourceService) put(ctx context.Context, session model.Session, in UploadInput, body io.Reader, action model.ResourceAction) (model.Resource, error) {
	// The bytes are written to a temporary file before the transaction, so a
	// slow upload does not hold the database write lock.
	staged, err := s.stage(ctx, body, MaxUploadBytes)
	if err != nil {
		return model.Resource{}, err
	}
	defer s.store.Discard(staged)
	isText := staged.IsText && staged.Size <= MaxTextBytes

	var result model.Resource
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.requireFolder(ctx, tx, session.WorkspaceID, in.ParentID); err != nil {
			return err
		}
		name, replaceID, err := s.resolveName(ctx, tx, session.WorkspaceID, in.ParentID, in.Name, model.ResourceFile, in.OnConflict, "")
		if err != nil {
			return err
		}
		mimeType := detectMime(name, in.ContentType, staged.Head)
		now := s.nowString()
		id := replaceID
		if replaceID != "" {
			if err := s.repo.UpdateContent(ctx, tx, session.WorkspaceID, id, mimeType, staged.Size, isText, now); err != nil {
				return err
			}
			if err := s.logEvent(ctx, tx, session, id, "", action, map[string]any{"name": name, "size": staged.Size, "replaced": true}, now); err != nil {
				return err
			}
		} else {
			id = model.NewID()
			r := model.Resource{
				ID: id, WorkspaceID: session.WorkspaceID, ParentID: in.ParentID, Kind: model.ResourceFile,
				Name: name, MimeType: mimeType, Size: staged.Size, IsText: isText, CreatedBy: session.MembershipID,
			}
			if err := s.repo.Insert(ctx, tx, r, nameKey(name), now); err != nil {
				return err
			}
			if err := s.logEvent(ctx, tx, session, id, "", action, map[string]any{"name": name, "size": staged.Size}, now); err != nil {
				return err
			}
		}
		path, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := s.store.Place(staged, path); err != nil {
			return err
		}
		result, err = s.repo.Get(ctx, tx, session.WorkspaceID, id)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return result, nil
}

// SaveContent replaces the text of a file. expectedRevision must match the
// stored revision, so a save based on an older read is rejected as stale.
// Saves by the same member within editMergeWindow of each other are logged
// as one 'edited' event.
func (s *ResourceService) SaveContent(ctx context.Context, session model.Session, id string, expectedRevision int64, body io.Reader) (model.Resource, error) {
	staged, err := s.stage(ctx, body, MaxTextBytes)
	if err != nil {
		return model.Resource{}, err
	}
	defer s.store.Discard(staged)
	if !staged.IsText {
		return model.Resource{}, fmt.Errorf("%w: content is not UTF-8 text", ErrInvalidInput)
	}

	var result model.Resource
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		r, err := s.live(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if r.Kind != model.ResourceFile {
			return fmt.Errorf("%w: not a file", ErrInvalidInput)
		}
		if r.Revision != expectedRevision {
			return fmt.Errorf("%w: revision %d, stored %d", ErrStale, expectedRevision, r.Revision)
		}
		nowTime := s.now().UTC()
		now := nowTime.Format(time.RFC3339)
		if err := s.repo.UpdateContent(ctx, tx, session.WorkspaceID, id, r.MimeType, staged.Size, true, now); err != nil {
			return err
		}
		fields := map[string]any{"name": r.Name, "size": staged.Size}
		last, err := s.repo.LastEvent(ctx, tx, session.WorkspaceID, id)
		switch {
		case err == nil && last.Action == model.ResourceEdited && last.ActorID == session.MembershipID &&
			nowTime.Sub(last.UpdatedAt) <= editMergeWindow:
			if err := s.repo.ExtendEvent(ctx, tx, session.WorkspaceID, last.ID, details(fields), now); err != nil {
				return err
			}
		case err == nil || errors.Is(err, sql.ErrNoRows):
			if err := s.logEvent(ctx, tx, session, id, "", model.ResourceEdited, fields, now); err != nil {
				return err
			}
		default:
			return err
		}
		path, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := s.store.Place(staged, path); err != nil {
			return err
		}
		result, err = s.repo.Get(ctx, tx, session.WorkspaceID, id)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return result, nil
}

// Open returns a file with its bytes; the caller closes the file.
func (s *ResourceService) Open(ctx context.Context, session model.Session, id string) (model.Resource, *os.File, error) {
	r, err := s.repo.Get(ctx, s.db, session.WorkspaceID, id)
	if err != nil {
		return model.Resource{}, nil, notFound(err, "resource")
	}
	if r.Kind != model.ResourceFile {
		return model.Resource{}, nil, fmt.Errorf("%w: not a file", ErrInvalidInput)
	}
	path, err := s.diskPath(ctx, s.db, session.WorkspaceID, id)
	if err != nil {
		return model.Resource{}, nil, err
	}
	f, err := s.store.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return model.Resource{}, nil, fmt.Errorf("%w: %s", ErrFileMissing, r.ID)
		}
		return model.Resource{}, nil, err
	}
	return r, f, nil
}

func (s *ResourceService) Rename(ctx context.Context, session model.Session, id, name string) (model.Resource, error) {
	var result model.Resource
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		r, err := s.live(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		newName, _, err := s.resolveName(ctx, tx, session.WorkspaceID, r.ParentID, name, r.Kind, ConflictFail, r.ID)
		if err != nil {
			return err
		}
		if newName == r.Name {
			result = r
			return nil
		}
		from, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		now := s.nowString()
		if err := s.repo.Place(ctx, tx, session.WorkspaceID, id, r.ParentID, newName, nameKey(newName), now); err != nil {
			return err
		}
		if err := s.logEvent(ctx, tx, session, id, "", model.ResourceRenamed, map[string]any{"name": newName, "from": r.Name}, now); err != nil {
			return err
		}
		to, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := tolerateMissing(s.store.Move(from, to)); err != nil {
			return err
		}
		result, err = s.repo.Get(ctx, tx, session.WorkspaceID, id)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return result, nil
}

type TransferInput struct {
	// ParentID is the target folder; "" is the top of the tree.
	ParentID   string
	OnConflict OnConflict
}

// checkTarget validates a move or copy target: a live folder that is not the
// item itself or inside it.
func (s *ResourceService) checkTarget(ctx context.Context, q database.DBTX, workspaceID string, r model.Resource, in TransferInput) error {
	if !in.OnConflict.valid() {
		return fmt.Errorf("%w: on_conflict", ErrInvalidInput)
	}
	if err := s.requireFolder(ctx, q, workspaceID, in.ParentID); err != nil {
		return err
	}
	if in.ParentID == "" || r.Kind != model.ResourceFolder {
		return nil
	}
	chain, err := s.repo.Ancestors(ctx, q, workspaceID, in.ParentID)
	if err != nil {
		return err
	}
	for _, a := range chain {
		if a.ID == r.ID {
			return fmt.Errorf("%w: a folder cannot go inside itself", ErrInvalidInput)
		}
	}
	return nil
}

// Move puts a live item into another folder. With ConflictReplace the file
// already there is moved to the trash.
func (s *ResourceService) Move(ctx context.Context, session model.Session, id string, in TransferInput) (model.Resource, error) {
	var result model.Resource
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		r, err := s.live(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := s.checkTarget(ctx, tx, session.WorkspaceID, r, in); err != nil {
			return err
		}
		if in.ParentID == r.ParentID {
			result = r
			return nil
		}
		name, replaceID, err := s.resolveName(ctx, tx, session.WorkspaceID, in.ParentID, r.Name, r.Kind, in.OnConflict, "")
		if err != nil {
			return err
		}
		now := s.nowString()
		if replaceID != "" {
			if err := s.trash(ctx, tx, session, replaceID, now); err != nil {
				return err
			}
		}
		fromPath, err := s.displayPath(ctx, tx, session.WorkspaceID, r.ParentID)
		if err != nil {
			return err
		}
		toPath, err := s.displayPath(ctx, tx, session.WorkspaceID, in.ParentID)
		if err != nil {
			return err
		}
		from, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := s.repo.Place(ctx, tx, session.WorkspaceID, id, in.ParentID, name, nameKey(name), now); err != nil {
			return err
		}
		if err := s.logEvent(ctx, tx, session, id, "", model.ResourceMoved,
			map[string]any{"name": name, "from": fromPath, "to": toPath}, now); err != nil {
			return err
		}
		to, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := tolerateMissing(s.store.Move(from, to)); err != nil {
			return err
		}
		result, err = s.repo.Get(ctx, tx, session.WorkspaceID, id)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return result, nil
}

// Copy duplicates a live item (a folder with everything in it) into a
// folder, which may be its own.
func (s *ResourceService) Copy(ctx context.Context, session model.Session, id string, in TransferInput) (model.Resource, error) {
	var result model.Resource
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		r, err := s.live(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := s.checkTarget(ctx, tx, session.WorkspaceID, r, in); err != nil {
			return err
		}
		name, replaceID, err := s.resolveName(ctx, tx, session.WorkspaceID, in.ParentID, r.Name, r.Kind, in.OnConflict, "")
		if err != nil {
			return err
		}
		if replaceID == r.ID {
			return fmt.Errorf("%w: a file cannot replace itself", ErrConflict)
		}
		now := s.nowString()
		if replaceID != "" {
			if err := s.trash(ctx, tx, session, replaceID, now); err != nil {
				return err
			}
		}
		subtree, err := s.repo.LiveSubtree(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		from, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		newIDs := map[string]string{}
		for _, item := range subtree {
			c := item
			c.ID = model.NewID()
			c.CreatedBy = session.MembershipID
			if item.ID == id {
				c.ParentID, c.Name = in.ParentID, name
			} else {
				c.ParentID = newIDs[item.ParentID]
			}
			newIDs[item.ID] = c.ID
			if err := s.repo.Insert(ctx, tx, c, nameKey(c.Name), now); err != nil {
				return err
			}
		}
		copyID := newIDs[id]
		sourcePath, err := s.displayPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := s.logEvent(ctx, tx, session, copyID, "", model.ResourceCopied,
			map[string]any{"name": name, "from": sourcePath}, now); err != nil {
			return err
		}
		to, err := s.diskPath(ctx, tx, session.WorkspaceID, copyID)
		if err != nil {
			return err
		}
		if err := s.store.Copy(from, to); err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if r.Kind == model.ResourceFile {
				return fmt.Errorf("%w: %s", ErrFileMissing, r.ID)
			}
			if err := s.store.MakeDir(to); err != nil {
				return err
			}
		}
		result, err = s.repo.Get(ctx, tx, session.WorkspaceID, copyID)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return result, nil
}

// MoveToTrash moves a live item, with everything in it, to the trash. Its
// task links stay; they show the item as trashed.
func (s *ResourceService) MoveToTrash(ctx context.Context, session model.Session, id string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := s.live(ctx, tx, session.WorkspaceID, id); err != nil {
			return err
		}
		return s.trash(ctx, tx, session, id, s.nowString())
	})
}

func (s *ResourceService) trash(ctx context.Context, tx *sql.Tx, session model.Session, id, now string) error {
	r, err := s.repo.Get(ctx, tx, session.WorkspaceID, id)
	if err != nil {
		return err
	}
	from, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
	if err != nil {
		return err
	}
	location, err := s.displayPath(ctx, tx, session.WorkspaceID, r.ParentID)
	if err != nil {
		return err
	}
	if err := s.repo.Trash(ctx, tx, session.WorkspaceID, id, now); err != nil {
		return err
	}
	if err := s.logEvent(ctx, tx, session, id, "", model.ResourceTrashed, map[string]any{"name": r.Name, "from": location}, now); err != nil {
		return err
	}
	to, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
	if err != nil {
		return err
	}
	return tolerateMissing(s.store.Move(from, to))
}

// trashRoot returns an item that was moved to the trash itself.
func (s *ResourceService) trashRoot(ctx context.Context, q database.DBTX, workspaceID, id string) (model.Resource, error) {
	r, err := s.repo.Get(ctx, q, workspaceID, id)
	if err != nil {
		return model.Resource{}, notFound(err, "resource")
	}
	if r.TrashedRootID != r.ID {
		return model.Resource{}, fmt.Errorf("%w: not in the trash", ErrNotFound)
	}
	return r, nil
}

// Restore takes an item out of the trash, back into its folder, or to the top
// of the tree when that folder is gone. A taken name gets a " (n)" suffix.
func (s *ResourceService) Restore(ctx context.Context, session model.Session, id string) (model.Resource, error) {
	var result model.Resource
	err := database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		r, err := s.trashRoot(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		parentID := r.ParentID
		if parentID != "" {
			if _, err := s.live(ctx, tx, session.WorkspaceID, parentID); err != nil {
				if !errors.Is(err, ErrNotFound) {
					return err
				}
				parentID = ""
			}
		}
		name, _, err := s.resolveName(ctx, tx, session.WorkspaceID, parentID, r.Name, r.Kind, ConflictKeepBoth, "")
		if err != nil {
			return err
		}
		from, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		now := s.nowString()
		// Renamed while still trashed, so the unique sibling index never sees
		// the old name come back.
		if err := s.repo.Place(ctx, tx, session.WorkspaceID, id, parentID, name, nameKey(name), now); err != nil {
			return err
		}
		if err := s.repo.Restore(ctx, tx, session.WorkspaceID, id, now); err != nil {
			return err
		}
		location, err := s.displayPath(ctx, tx, session.WorkspaceID, parentID)
		if err != nil {
			return err
		}
		if err := s.logEvent(ctx, tx, session, id, "", model.ResourceRestored, map[string]any{"name": name, "to": location}, now); err != nil {
			return err
		}
		to, err := s.diskPath(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		if err := tolerateMissing(s.store.Move(from, to)); err != nil {
			return err
		}
		if err := s.store.Remove([]string{session.WorkspaceID, trashDirName, id}); err != nil {
			return err
		}
		result, err = s.repo.Get(ctx, tx, session.WorkspaceID, id)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return result, nil
}

// Purge deletes a trashed item for good: its bytes are removed and its rows
// stay as tombstones, so the activity log still names it.
func (s *ResourceService) Purge(ctx context.Context, session model.Session, id string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		r, err := s.trashRoot(ctx, tx, session.WorkspaceID, id)
		if err != nil {
			return err
		}
		return s.purge(ctx, tx, session, r, s.nowString())
	})
}

// EmptyTrash purges everything in the trash.
func (s *ResourceService) EmptyTrash(ctx context.Context, session model.Session) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		items, err := s.repo.ListTrash(ctx, tx, session.WorkspaceID)
		if err != nil {
			return err
		}
		now := s.nowString()
		for _, r := range items {
			if err := s.purge(ctx, tx, session, r, now); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *ResourceService) purge(ctx context.Context, tx *sql.Tx, session model.Session, r model.Resource, now string) error {
	if err := s.repo.CloseLinksOfTrashGroup(ctx, tx, session.WorkspaceID, r.ID, now); err != nil {
		return err
	}
	if err := s.repo.Purge(ctx, tx, session.WorkspaceID, r.ID, now); err != nil {
		return err
	}
	if err := s.logEvent(ctx, tx, session, r.ID, "", model.ResourcePurged, map[string]any{"name": r.Name}, now); err != nil {
		return err
	}
	return s.store.Remove([]string{session.WorkspaceID, trashDirName, r.ID})
}

func (s *ResourceService) requireTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) (model.Task, error) {
	t, err := s.tasks.Get(ctx, q, workspaceID, taskID)
	if err != nil {
		return model.Task{}, notFound(err, "task")
	}
	return t, nil
}

// Link links a live resource to a task. Linking twice is a no-op.
func (s *ResourceService) Link(ctx context.Context, session model.Session, taskID, resourceID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		task, err := s.requireTask(ctx, tx, session.WorkspaceID, taskID)
		if err != nil {
			return err
		}
		r, err := s.live(ctx, tx, session.WorkspaceID, resourceID)
		if err != nil {
			return err
		}
		now := s.nowString()
		changed, err := s.repo.Link(ctx, tx, session.WorkspaceID, taskID, resourceID, session.MembershipID, now)
		if err != nil || !changed {
			return err
		}
		return s.logEvent(ctx, tx, session, resourceID, taskID, model.ResourceLinked,
			map[string]any{"name": r.Name, "task_title": task.Title}, now)
	})
}

// Unlink removes a link; removing a missing link is a no-op.
func (s *ResourceService) Unlink(ctx context.Context, session model.Session, taskID, resourceID string) error {
	return database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		task, err := s.requireTask(ctx, tx, session.WorkspaceID, taskID)
		if err != nil {
			return err
		}
		r, err := s.repo.Get(ctx, tx, session.WorkspaceID, resourceID)
		if err != nil {
			return notFound(err, "resource")
		}
		now := s.nowString()
		changed, err := s.repo.Unlink(ctx, tx, session.WorkspaceID, taskID, resourceID, now)
		if err != nil || !changed {
			return err
		}
		return s.logEvent(ctx, tx, session, resourceID, taskID, model.ResourceUnlinked,
			map[string]any{"name": r.Name, "task_title": task.Title}, now)
	})
}

func (s *ResourceService) TaskResources(ctx context.Context, session model.Session, taskID string) ([]model.Resource, error) {
	if _, err := s.requireTask(ctx, s.db, session.WorkspaceID, taskID); err != nil {
		return nil, err
	}
	return s.repo.ListByTask(ctx, s.db, session.WorkspaceID, taskID)
}

func (s *ResourceService) ResourceTasks(ctx context.Context, session model.Session, resourceID string) ([]model.LinkedTask, error) {
	if _, err := s.Get(ctx, session, resourceID); err != nil {
		return nil, err
	}
	return s.repo.ListTasks(ctx, s.db, session.WorkspaceID, resourceID)
}

type EventQuery struct {
	// ResourceID or ProjectID narrows the log; both empty lists the
	// workspace.
	ResourceID string
	ProjectID  string
	// Before is the id of the oldest event already shown.
	Before string
	Limit  int
}

// Events returns a page of the activity log, newest first. Purged resources
// keep their history, so an unknown id simply has no events.
func (s *ResourceService) Events(ctx context.Context, session model.Session, in EventQuery) ([]model.ResourceEvent, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = defaultEventPage
	}
	if limit > maxEventPage {
		limit = maxEventPage
	}
	return s.repo.ListEvents(ctx, s.db, session.WorkspaceID,
		model.ResourceEventScope{ResourceID: in.ResourceID, ProjectID: in.ProjectID}, in.Before, limit)
}
