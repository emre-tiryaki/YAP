package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
	"github.com/emre-tiryaki/YAP/backend/internal/storage"
)

type resourceEnv struct {
	db      *sql.DB
	session model.Session
	svc     *ResourceService
	root    string
	ctx     context.Context
}

func newResourceEnv(t *testing.T) resourceEnv {
	t.Helper()
	db, session := newTestDB(t)
	root := t.TempDir()
	store, err := storage.NewFS(root)
	if err != nil {
		t.Fatalf("storage.NewFS() error = %v", err)
	}
	svc := NewResourceService(db, repository.NewResourceRepository(), repository.NewTaskRepository(), store)
	return resourceEnv{db: db, session: session, svc: svc, root: root, ctx: context.Background()}
}

// disk returns the path of a live item's bytes, from the workspace dir down.
func (e resourceEnv) disk(parts ...string) string {
	return filepath.Join(append([]string{e.root, e.session.WorkspaceID}, parts...)...)
}

func (e resourceEnv) folder(t *testing.T, parentID, name string) model.Resource {
	t.Helper()
	r, err := e.svc.CreateFolder(e.ctx, e.session, parentID, name)
	if err != nil {
		t.Fatalf("CreateFolder(%q) error = %v", name, err)
	}
	return r
}

func (e resourceEnv) upload(t *testing.T, parentID, name, content string) model.Resource {
	t.Helper()
	r, err := e.svc.Upload(e.ctx, e.session, UploadInput{ParentID: parentID, Name: name, OnConflict: ConflictFail}, strings.NewReader(content))
	if err != nil {
		t.Fatalf("Upload(%q) error = %v", name, err)
	}
	return r
}

func (e resourceEnv) read(t *testing.T, id string) string {
	t.Helper()
	_, f, err := e.svc.Open(e.ctx, e.session, id)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}

func (e resourceEnv) actions(t *testing.T, scope EventQuery) []model.ResourceAction {
	t.Helper()
	events, err := e.svc.Events(e.ctx, e.session, scope)
	if err != nil {
		t.Fatalf("Events() error = %v", err)
	}
	out := make([]model.ResourceAction, len(events))
	for i, ev := range events {
		out[i] = ev.Action
	}
	return out
}

func wantExists(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	if got := err == nil; got != want {
		t.Errorf("exists(%s) = %v, want %v (err %v)", path, got, want, err)
	}
}

func TestResourceNameRules(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		parent  bool
		wantErr error
	}{
		{"plain", "notes.txt", false, nil},
		{"trimmed", "  notes.txt ", false, nil},
		{"empty", "  ", false, ErrInvalidInput},
		{"dot", ".", false, ErrInvalidInput},
		{"dot dot", "..", false, ErrInvalidInput},
		{"slash", "a/b", false, ErrInvalidInput},
		{"backslash", `a\b`, false, ErrInvalidInput},
		{"colon", "a:b", false, ErrInvalidInput},
		{"control", "a\tb", false, ErrInvalidInput},
		{"too long", strings.Repeat("a", 256), false, ErrInvalidInput},
		{"reserved at top", ".YAP-trash", false, ErrInvalidInput},
		{"reserved name allowed in a folder", ".yap-trash", true, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newResourceEnv(t)
			parent := ""
			if tt.parent {
				parent = e.folder(t, "", "dir").ID
			}
			_, err := e.svc.CreateFolder(e.ctx, e.session, parent, tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateFolder(%q) error = %v, want %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestSiblingNamesAreUniqueCaseInsensitively(t *testing.T) {
	e := newResourceEnv(t)
	e.upload(t, "", "Report.md", "x")
	if _, err := e.svc.CreateFolder(e.ctx, e.session, "", "report.MD"); !errors.Is(err, ErrConflict) {
		t.Fatalf("CreateFolder(same name) error = %v, want ErrConflict", err)
	}
	// Another folder may reuse the name.
	dir := e.folder(t, "", "dir")
	e.upload(t, dir.ID, "report.md", "y")
}

func TestUploadConflictPolicies(t *testing.T) {
	tests := []struct {
		name       string
		policy     OnConflict
		wantErr    error
		wantName   string
		wantSameID bool
		wantText   string
	}{
		{"fail", ConflictFail, ErrConflict, "", false, ""},
		{"keep both", ConflictKeepBoth, nil, "Notes (1).txt", false, "new"},
		{"replace", ConflictReplace, nil, "notes.txt", true, "new"},
		{"unknown", OnConflict("merge"), ErrInvalidInput, "", false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newResourceEnv(t)
			old := e.upload(t, "", "notes.txt", "old")
			got, err := e.svc.Upload(e.ctx, e.session, UploadInput{Name: "Notes.txt", OnConflict: tt.policy}, strings.NewReader("new"))
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Upload() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if got.Name != tt.wantName {
				t.Errorf("name = %q, want %q", got.Name, tt.wantName)
			}
			if (got.ID == old.ID) != tt.wantSameID {
				t.Errorf("same id = %v, want %v", got.ID == old.ID, tt.wantSameID)
			}
			if tt.wantSameID && got.Revision != 2 {
				t.Errorf("revision = %d, want 2", got.Revision)
			}
			if text := e.read(t, got.ID); text != tt.wantText {
				t.Errorf("content = %q, want %q", text, tt.wantText)
			}
			wantExists(t, e.disk(tt.wantName), true)
		})
	}
}

func TestReplaceRejectsFolders(t *testing.T) {
	e := newResourceEnv(t)
	e.folder(t, "", "docs")
	_, err := e.svc.Upload(e.ctx, e.session, UploadInput{Name: "docs", OnConflict: ConflictReplace}, strings.NewReader("x"))
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Upload(replace folder) error = %v, want ErrConflict", err)
	}
}

func TestUploadDetectsTextAndType(t *testing.T) {
	e := newResourceEnv(t)
	tests := []struct {
		name, content, header string
		wantText              bool
		wantMime              string
	}{
		{"a.md", "# hi", "", true, "text/markdown"},
		{"a.json", `{"a":1}`, "", true, "application/json"},
		{"noext", "plain words", "", true, "text/plain"},
		{"blob", "\x00\x01\x02", "application/x-custom", false, "application/x-custom"},
		{"img.png", "\x89PNG\r\n\x1a\n", "", false, "image/png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := e.svc.Upload(e.ctx, e.session, UploadInput{Name: tt.name, OnConflict: ConflictFail, ContentType: tt.header}, strings.NewReader(tt.content))
			if err != nil {
				t.Fatalf("Upload() error = %v", err)
			}
			if r.IsText != tt.wantText {
				t.Errorf("IsText = %v, want %v", r.IsText, tt.wantText)
			}
			if r.MimeType != tt.wantMime {
				t.Errorf("MimeType = %q, want %q", r.MimeType, tt.wantMime)
			}
		})
	}
}

func TestSaveContentChecksRevision(t *testing.T) {
	e := newResourceEnv(t)
	f, err := e.svc.CreateFile(e.ctx, e.session, "", "todo.txt")
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	if !f.IsText || f.Revision != 1 || f.Size != 0 {
		t.Fatalf("new file = %+v, want empty text revision 1", f)
	}

	saved, err := e.svc.SaveContent(e.ctx, e.session, f.ID, 1, strings.NewReader("first"))
	if err != nil {
		t.Fatalf("SaveContent() error = %v", err)
	}
	if saved.Revision != 2 || saved.Size != 5 {
		t.Errorf("saved = rev %d size %d, want rev 2 size 5", saved.Revision, saved.Size)
	}
	if _, err := e.svc.SaveContent(e.ctx, e.session, f.ID, 1, strings.NewReader("stale")); !errors.Is(err, ErrStale) {
		t.Fatalf("SaveContent(old revision) error = %v, want ErrStale", err)
	}
	if got := e.read(t, f.ID); got != "first" {
		t.Errorf("content = %q, want %q", got, "first")
	}
	if _, err := e.svc.SaveContent(e.ctx, e.session, f.ID, 2, strings.NewReader("bad\x00")); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("SaveContent(binary) error = %v, want ErrInvalidInput", err)
	}
}

func TestSaveContentMergesEditsWithinWindow(t *testing.T) {
	e := newResourceEnv(t)
	clock := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	e.svc.now = func() time.Time { return clock }
	f, err := e.svc.CreateFile(e.ctx, e.session, "", "a.txt")
	if err != nil {
		t.Fatalf("CreateFile() error = %v", err)
	}
	save := func(rev int64, text string) {
		t.Helper()
		if _, err := e.svc.SaveContent(e.ctx, e.session, f.ID, rev, strings.NewReader(text)); err != nil {
			t.Fatalf("SaveContent() error = %v", err)
		}
	}
	save(1, "a")
	clock = clock.Add(5 * time.Minute)
	save(2, "ab")
	clock = clock.Add(9 * time.Minute) // 9 minutes after the last save: still merged
	save(3, "abc")
	clock = clock.Add(11 * time.Minute)
	save(4, "abcd")

	got := e.actions(t, EventQuery{ResourceID: f.ID})
	want := []model.ResourceAction{model.ResourceEdited, model.ResourceEdited, model.ResourceCreated}
	if !equalActions(got, want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
}

func equalActions(a, b []model.ResourceAction) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRenameAndMoveFollowOnDisk(t *testing.T) {
	e := newResourceEnv(t)
	docs := e.folder(t, "", "docs")
	archive := e.folder(t, "", "archive")
	f := e.upload(t, docs.ID, "a.txt", "hello")

	if _, err := e.svc.Rename(e.ctx, e.session, f.ID, "A.txt"); err != nil {
		t.Fatalf("Rename(case only) error = %v", err)
	}
	wantExists(t, e.disk("docs", "A.txt"), true)

	if _, err := e.svc.Move(e.ctx, e.session, docs.ID, TransferInput{ParentID: archive.ID, OnConflict: ConflictFail}); err != nil {
		t.Fatalf("Move() error = %v", err)
	}
	wantExists(t, e.disk("archive", "docs", "A.txt"), true)
	wantExists(t, e.disk("docs"), false)
	if got := e.read(t, f.ID); got != "hello" {
		t.Errorf("content after move = %q", got)
	}
}

func TestMoveRejectsBadTargets(t *testing.T) {
	e := newResourceEnv(t)
	a := e.folder(t, "", "a")
	b := e.folder(t, a.ID, "b")
	f := e.upload(t, "", "f.txt", "x")
	tests := []struct {
		name    string
		id      string
		target  string
		wantErr error
	}{
		{"into itself", a.ID, a.ID, ErrInvalidInput},
		{"into its child", a.ID, b.ID, ErrInvalidInput},
		{"into a file", a.ID, f.ID, ErrInvalidInput},
		{"into a missing folder", f.ID, model.NewID(), ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.svc.Move(e.ctx, e.session, tt.id, TransferInput{ParentID: tt.target, OnConflict: ConflictFail})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Move() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestMoveReplaceTrashesExisting(t *testing.T) {
	e := newResourceEnv(t)
	dst := e.folder(t, "", "dst")
	existing := e.upload(t, dst.ID, "a.txt", "old")
	f := e.upload(t, "", "a.txt", "new")

	if _, err := e.svc.Move(e.ctx, e.session, f.ID, TransferInput{ParentID: dst.ID, OnConflict: ConflictReplace}); err != nil {
		t.Fatalf("Move(replace) error = %v", err)
	}
	trash, err := e.svc.Trash(e.ctx, e.session)
	if err != nil || len(trash) != 1 || trash[0].ID != existing.ID {
		t.Fatalf("trash = %v (err %v), want the replaced file", trash, err)
	}
	if got := e.read(t, f.ID); got != "new" {
		t.Errorf("moved content = %q", got)
	}
}

func TestCopyDuplicatesSubtree(t *testing.T) {
	e := newResourceEnv(t)
	docs := e.folder(t, "", "docs")
	sub := e.folder(t, docs.ID, "sub")
	e.upload(t, sub.ID, "a.txt", "hello")

	c, err := e.svc.Copy(e.ctx, e.session, docs.ID, TransferInput{ParentID: "", OnConflict: ConflictKeepBoth})
	if err != nil {
		t.Fatalf("Copy() error = %v", err)
	}
	if c.Name != "docs (1)" || c.ID == docs.ID {
		t.Fatalf("copy = %q %s, want a new docs (1)", c.Name, c.ID)
	}
	wantExists(t, e.disk("docs (1)", "sub", "a.txt"), true)

	all, err := e.svc.List(e.ctx, e.session)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(all) != 6 {
		t.Errorf("live resources = %d, want 6", len(all))
	}
	if _, err := e.svc.Copy(e.ctx, e.session, docs.ID, TransferInput{ParentID: sub.ID, OnConflict: ConflictFail}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("Copy(into own child) error = %v, want ErrInvalidInput", err)
	}
}

func TestTrashRestoreAndPurge(t *testing.T) {
	e := newResourceEnv(t)
	docs := e.folder(t, "", "docs")
	f := e.upload(t, docs.ID, "a.txt", "hello")

	if err := e.svc.MoveToTrash(e.ctx, e.session, docs.ID); err != nil {
		t.Fatalf("MoveToTrash() error = %v", err)
	}
	wantExists(t, e.disk("docs"), false)
	wantExists(t, e.disk(trashDirName, docs.ID, "docs", "a.txt"), true)
	if all, _ := e.svc.List(e.ctx, e.session); len(all) != 0 {
		t.Errorf("live resources after trash = %d, want 0", len(all))
	}
	// Its contents are not listed on their own and cannot be changed.
	if trash, _ := e.svc.Trash(e.ctx, e.session); len(trash) != 1 {
		t.Errorf("trash = %d items, want 1", len(trash))
	}
	if _, err := e.svc.Rename(e.ctx, e.session, f.ID, "b.txt"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Rename(trashed) error = %v, want ErrNotFound", err)
	}
	if got := e.read(t, f.ID); got != "hello" {
		t.Errorf("trashed content = %q", got)
	}
	// A new item can take the trashed name; restoring then picks another.
	e.folder(t, "", "docs")
	restored, err := e.svc.Restore(e.ctx, e.session, docs.ID)
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if restored.Name != "docs (1)" || restored.TrashedAt != nil {
		t.Errorf("restored = %q trashed %v, want docs (1), live", restored.Name, restored.TrashedAt)
	}
	wantExists(t, e.disk("docs (1)", "a.txt"), true)
	wantExists(t, e.disk(trashDirName, docs.ID), false)

	if err := e.svc.Purge(e.ctx, e.session, docs.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Purge(live) error = %v, want ErrNotFound", err)
	}
	if err := e.svc.MoveToTrash(e.ctx, e.session, docs.ID); err != nil {
		t.Fatalf("MoveToTrash() error = %v", err)
	}
	if err := e.svc.Purge(e.ctx, e.session, docs.ID); err != nil {
		t.Fatalf("Purge() error = %v", err)
	}
	wantExists(t, e.disk(trashDirName, docs.ID), false)
	if _, err := e.svc.Get(e.ctx, e.session, f.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get(purged child) error = %v, want ErrNotFound", err)
	}
	// The log keeps the history of the purged folder.
	got := e.actions(t, EventQuery{ResourceID: docs.ID})
	want := []model.ResourceAction{
		model.ResourcePurged, model.ResourceTrashed, model.ResourceRestored,
		model.ResourceTrashed, model.ResourceUploaded, model.ResourceCreated,
	}
	if !equalActions(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestRestoreGoesToTopWhenFolderIsGone(t *testing.T) {
	e := newResourceEnv(t)
	docs := e.folder(t, "", "docs")
	f := e.upload(t, docs.ID, "a.txt", "x")
	if err := e.svc.MoveToTrash(e.ctx, e.session, f.ID); err != nil {
		t.Fatalf("MoveToTrash(file) error = %v", err)
	}
	if err := e.svc.MoveToTrash(e.ctx, e.session, docs.ID); err != nil {
		t.Fatalf("MoveToTrash(folder) error = %v", err)
	}
	// Each was trashed on its own, so both are listed.
	if trash, _ := e.svc.Trash(e.ctx, e.session); len(trash) != 2 {
		t.Fatalf("trash = %d items, want 2", len(trash))
	}
	restored, err := e.svc.Restore(e.ctx, e.session, f.ID)
	if err != nil {
		t.Fatalf("Restore() error = %v", err)
	}
	if restored.ParentID != "" {
		t.Errorf("parent = %q, want top of the tree", restored.ParentID)
	}
	wantExists(t, e.disk("a.txt"), true)
}

func TestEmptyTrash(t *testing.T) {
	e := newResourceEnv(t)
	a := e.upload(t, "", "a.txt", "x")
	b := e.folder(t, "", "b")
	for _, id := range []string{a.ID, b.ID} {
		if err := e.svc.MoveToTrash(e.ctx, e.session, id); err != nil {
			t.Fatalf("MoveToTrash() error = %v", err)
		}
	}
	if err := e.svc.EmptyTrash(e.ctx, e.session); err != nil {
		t.Fatalf("EmptyTrash() error = %v", err)
	}
	if trash, _ := e.svc.Trash(e.ctx, e.session); len(trash) != 0 {
		t.Errorf("trash = %d items, want 0", len(trash))
	}
	wantExists(t, e.disk(trashDirName, a.ID), false)
}

func TestMissingBytesAreReported(t *testing.T) {
	e := newResourceEnv(t)
	f := e.upload(t, "", "a.txt", "x")
	if err := os.Remove(e.disk("a.txt")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := e.svc.Open(e.ctx, e.session, f.ID); !errors.Is(err, ErrFileMissing) {
		t.Fatalf("Open() error = %v, want ErrFileMissing", err)
	}
	// The tree still follows the database.
	if _, err := e.svc.Rename(e.ctx, e.session, f.ID, "b.txt"); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}
	if err := e.svc.MoveToTrash(e.ctx, e.session, f.ID); err != nil {
		t.Fatalf("MoveToTrash() error = %v", err)
	}
}

func TestUploadTooLarge(t *testing.T) {
	e := newResourceEnv(t)
	body := io.LimitReader(zeroReader{}, MaxUploadBytes+1)
	if _, err := e.svc.Upload(e.ctx, e.session, UploadInput{Name: "big.bin", OnConflict: ConflictFail}, body); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Upload() error = %v, want ErrTooLarge", err)
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}

func newTask(t *testing.T, e resourceEnv) (projectID string, task model.Task) {
	t.Helper()
	projectRepo := repository.NewProjectRepository()
	project, err := NewProjectService(e.db, projectRepo, repository.NewSettingsRepository()).Create(e.ctx, e.session.WorkspaceID, CreateProjectInput{Name: "P"})
	if err != nil {
		t.Fatalf("Create project error = %v", err)
	}
	task, err = NewTaskService(e.db, projectRepo, repository.NewTaskRepository(), repository.NewTaskActivityRepository()).Create(e.ctx, e.session, project.ID, CreateTaskInput{Title: "T"})
	if err != nil {
		t.Fatalf("Create task error = %v", err)
	}
	return project.ID, task
}

func TestTaskLinks(t *testing.T) {
	e := newResourceEnv(t)
	projectID, task := newTask(t, e)
	f := e.upload(t, "", "spec.md", "x")

	for range 2 { // linking twice is a no-op
		if err := e.svc.Link(e.ctx, e.session, task.ID, f.ID); err != nil {
			t.Fatalf("Link() error = %v", err)
		}
	}
	linked, err := e.svc.TaskResources(e.ctx, e.session, task.ID)
	if err != nil || len(linked) != 1 {
		t.Fatalf("TaskResources() = %d items (err %v), want 1", len(linked), err)
	}
	tasks, err := e.svc.ResourceTasks(e.ctx, e.session, f.ID)
	if err != nil || len(tasks) != 1 || tasks[0].ProjectID != projectID {
		t.Fatalf("ResourceTasks() = %v (err %v)", tasks, err)
	}

	// A trashed resource stays linked and shows as trashed.
	if err := e.svc.MoveToTrash(e.ctx, e.session, f.ID); err != nil {
		t.Fatalf("MoveToTrash() error = %v", err)
	}
	linked, _ = e.svc.TaskResources(e.ctx, e.session, task.ID)
	if len(linked) != 1 || linked[0].TrashedAt == nil {
		t.Fatalf("linked after trash = %+v, want one trashed item", linked)
	}
	if err := e.svc.Link(e.ctx, e.session, task.ID, f.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Link(trashed) error = %v, want ErrNotFound", err)
	}
	// Purging closes the link.
	if err := e.svc.Purge(e.ctx, e.session, f.ID); err != nil {
		t.Fatalf("Purge() error = %v", err)
	}
	if linked, _ = e.svc.TaskResources(e.ctx, e.session, task.ID); len(linked) != 0 {
		t.Errorf("linked after purge = %d, want 0", len(linked))
	}

	got := e.actions(t, EventQuery{ProjectID: projectID})
	want := []model.ResourceAction{model.ResourceLinked}
	if !equalActions(got, want) {
		t.Errorf("project events = %v, want %v", got, want)
	}
}

func TestUnlinkAndRelink(t *testing.T) {
	e := newResourceEnv(t)
	_, task := newTask(t, e)
	f := e.upload(t, "", "a.txt", "x")
	if err := e.svc.Link(e.ctx, e.session, task.ID, f.ID); err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	for range 2 { // unlinking twice is a no-op
		if err := e.svc.Unlink(e.ctx, e.session, task.ID, f.ID); err != nil {
			t.Fatalf("Unlink() error = %v", err)
		}
	}
	if err := e.svc.Link(e.ctx, e.session, task.ID, f.ID); err != nil {
		t.Fatalf("Link() again error = %v", err)
	}
	got := e.actions(t, EventQuery{ResourceID: f.ID})
	want := []model.ResourceAction{model.ResourceLinked, model.ResourceUnlinked, model.ResourceLinked, model.ResourceUploaded}
	if !equalActions(got, want) {
		t.Errorf("events = %v, want %v", got, want)
	}
}

func TestDeletingTaskClosesLinks(t *testing.T) {
	e := newResourceEnv(t)
	_, task := newTask(t, e)
	f := e.upload(t, "", "a.txt", "x")
	if err := e.svc.Link(e.ctx, e.session, task.ID, f.ID); err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	taskSvc := NewTaskService(e.db, repository.NewProjectRepository(), repository.NewTaskRepository(), repository.NewTaskActivityRepository())
	if err := taskSvc.Delete(e.ctx, e.session, task.ID); err != nil {
		t.Fatalf("Delete task error = %v", err)
	}
	var open int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM task_resources WHERE removed_at IS NULL`).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 0 {
		t.Errorf("open links = %d, want 0", open)
	}
}

func TestEventsPaging(t *testing.T) {
	e := newResourceEnv(t)
	for _, name := range []string{"a", "b", "c"} {
		e.folder(t, "", name)
	}
	first, err := e.svc.Events(e.ctx, e.session, EventQuery{Limit: 2})
	if err != nil || len(first) != 2 {
		t.Fatalf("first page = %d (err %v), want 2", len(first), err)
	}
	rest, err := e.svc.Events(e.ctx, e.session, EventQuery{Limit: 2, Before: first[1].ID})
	if err != nil || len(rest) != 1 {
		t.Fatalf("second page = %d (err %v), want 1", len(rest), err)
	}
	if first[0].ActorName != "tester" {
		t.Errorf("actor = %q, want tester", first[0].ActorName)
	}
}

func TestFreeName(t *testing.T) {
	taken := map[string]string{"a.txt": "1", "a (1).txt": "2", ".env": "3", "dir": "4"}
	tests := []struct {
		name string
		kind model.ResourceKind
		want string
	}{
		{"b.txt", model.ResourceFile, "b.txt"},
		{"A.TXT", model.ResourceFile, "A (2).TXT"},
		{".env", model.ResourceFile, ".env (1)"},
		{"dir", model.ResourceFolder, "dir (1)"},
	}
	for _, tt := range tests {
		if got := freeName(tt.name, tt.kind, taken); got != tt.want {
			t.Errorf("freeName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
