package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"slices"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

type accountEnv struct {
	resourceEnv
	svc       *AccountService
	projects  *ProjectService
	tasks     *TaskService
	comments  *CommentService
	settings  *SettingsService
	workspace *WorkspaceService
}

func newAccountEnv(t *testing.T) accountEnv {
	t.Helper()
	re := newResourceEnv(t)
	projectRepo := repository.NewProjectRepository()
	taskRepo := repository.NewTaskRepository()
	store := re.svc.store
	return accountEnv{
		resourceEnv: re,
		svc: NewAccountService(re.db, repository.NewAccountRepository(), repository.NewResourceRepository(),
			repository.NewWorkspaceRepository(), repository.NewSettingsRepository(), store, "tester"),
		projects:  NewProjectService(re.db, projectRepo),
		tasks:     NewTaskService(re.db, projectRepo, taskRepo),
		comments:  NewCommentService(re.db, taskRepo, repository.NewCommentRepository()),
		settings:  NewSettingsService(re.db, repository.NewSettingsRepository()),
		workspace: NewWorkspaceService(re.db, repository.NewWorkspaceRepository()),
	}
}

// seed fills the workspace with a bit of everything: a live and a deleted
// project, a task with a comment and a linked file, a folder, a trashed file
// and changed settings and names.
func (e accountEnv) seed(t *testing.T) {
	t.Helper()
	project, err := e.projects.Create(e.ctx, e.session.WorkspaceID, CreateProjectInput{Name: "Kept"})
	if err != nil {
		t.Fatalf("Create project error = %v", err)
	}
	gone, err := e.projects.Create(e.ctx, e.session.WorkspaceID, CreateProjectInput{Name: "Gone"})
	if err != nil {
		t.Fatalf("Create project error = %v", err)
	}
	if err := e.projects.Delete(e.ctx, e.session.WorkspaceID, gone.ID); err != nil {
		t.Fatalf("Delete project error = %v", err)
	}
	task, err := e.tasks.Create(e.ctx, e.session, project.ID, CreateTaskInput{Title: "T"})
	if err != nil {
		t.Fatalf("Create task error = %v", err)
	}
	if _, err := e.comments.Create(e.ctx, e.session, task.ID, "hello"); err != nil {
		t.Fatalf("Create comment error = %v", err)
	}
	docs := e.folder(t, "", "docs")
	spec := e.upload(t, docs.ID, "spec.md", "# spec")
	e.folder(t, docs.ID, "empty")
	if err := e.resourceEnv.svc.Link(e.ctx, e.session, task.ID, spec.ID); err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	old := e.upload(t, "", "old.txt", "old")
	if err := e.resourceEnv.svc.MoveToTrash(e.ctx, e.session, old.ID); err != nil {
		t.Fatalf("MoveToTrash() error = %v", err)
	}
	off, delay := false, 5000
	if _, err := e.settings.Update(e.ctx, e.session, UpdateSettingsInput{AutosaveEnabled: &off, AutosaveDelayMs: &delay}); err != nil {
		t.Fatalf("Update settings error = %v", err)
	}
	user, ws := "Someone", "Work"
	if _, err := e.workspace.Update(e.ctx, e.session, UpdateProfileInput{DisplayName: &user, WorkspaceName: &ws}); err != nil {
		t.Fatalf("Update profile error = %v", err)
	}
}

func (e accountEnv) export(t *testing.T) (exportDocument, map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	if err := e.svc.Export(e.ctx, e.session, &buf); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	var doc exportDocument
	files := map[string]string{}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("open %s: %v", f.Name, err)
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("read %s: %v", f.Name, err)
		}
		if f.Name == exportDataName {
			if err := json.Unmarshal(b, &doc); err != nil {
				t.Fatalf("decode data.json: %v", err)
			}
			continue
		}
		files[f.Name] = string(b)
	}
	return doc, files
}

func column(rows []map[string]any, name string) []string {
	out := []string{}
	for _, r := range rows {
		if v, ok := r[name].(string); ok {
			out = append(out, v)
		}
	}
	slices.Sort(out)
	return out
}

func TestExport(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)

	doc, files := e.export(t)

	if doc.Format != ExportFormat || doc.Version != ExportVersion || doc.ExportedAt == "" {
		t.Errorf("header = %q v%d at %q", doc.Format, doc.Version, doc.ExportedAt)
	}
	tests := []struct {
		table  string
		column string
		want   []string
	}{
		{"users", "display_name", []string{"Someone"}},
		{"workspaces", "name", []string{"Work"}},
		{"projects", "name", []string{"Kept"}},
		{"plans", "name", []string{"Main"}},
		{"tasks", "title", []string{"T"}},
		{"task_comments", "body", []string{"hello"}},
		{"resources", "name", []string{"docs", "empty", "spec.md"}},
		{"user_settings", "key", []string{settingAutosaveDelayMs, settingAutosaveEnabled}},
	}
	for _, tt := range tests {
		t.Run(tt.table, func(t *testing.T) {
			rows, ok := doc.Tables[tt.table]
			if !ok {
				t.Fatalf("table %s missing", tt.table)
			}
			if got := column(rows, tt.column); !slices.Equal(got, tt.want) {
				t.Errorf("%s.%s = %v, want %v", tt.table, tt.column, got, tt.want)
			}
		})
	}
	if n := len(doc.Tables["task_resources"]); n != 1 {
		t.Errorf("task_resources = %d rows, want 1", n)
	}
	if n := len(doc.Tables["statuses"]); n != 3 {
		t.Errorf("statuses = %d rows, want the 3 default statuses of the live plan", n)
	}
	if n := len(doc.Tables["resource_events"]); n == 0 {
		t.Error("resource_events is empty")
	}

	wantFiles := map[string]string{
		"files/docs/":        "",
		"files/docs/empty/":  "",
		"files/docs/spec.md": "# spec",
	}
	if len(files) != len(wantFiles) {
		t.Errorf("archive entries = %v, want %v", files, wantFiles)
	}
	for name, content := range wantFiles {
		if got, ok := files[name]; !ok || got != content {
			t.Errorf("entry %s = %q (present %v), want %q", name, got, ok, content)
		}
	}
	if len(doc.MissingFiles) != 0 {
		t.Errorf("missing files = %v, want none", doc.MissingFiles)
	}
}

func TestExportReportsMissingFiles(t *testing.T) {
	e := newAccountEnv(t)
	docs := e.folder(t, "", "docs")
	e.upload(t, docs.ID, "a.txt", "a")
	e.upload(t, "", "b.txt", "b")
	if err := os.Remove(e.disk("docs", "a.txt")); err != nil {
		t.Fatal(err)
	}

	doc, files := e.export(t)

	if !slices.Equal(doc.MissingFiles, []string{"docs/a.txt"}) {
		t.Errorf("missing files = %v, want [docs/a.txt]", doc.MissingFiles)
	}
	if files["files/b.txt"] != "b" {
		t.Errorf("files/b.txt = %q, want %q", files["files/b.txt"], "b")
	}
	if _, ok := files["files/docs/a.txt"]; ok {
		t.Error("missing file has an archive entry")
	}
}

func TestReset(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)

	if err := e.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	t.Run("no live rows are left", func(t *testing.T) {
		live := map[string]string{
			"projects":        "deleted_at IS NULL",
			"plans":           "deleted_at IS NULL",
			"statuses":        "deleted_at IS NULL",
			"tasks":           "deleted_at IS NULL",
			"task_comments":   "deleted_at IS NULL",
			"task_assignees":  "removed_at IS NULL",
			"task_resources":  "removed_at IS NULL",
			"resources":       "deleted_at IS NULL",
			"resource_events": "deleted_at IS NULL",
		}
		for table, cond := range live {
			var liveRows, allRows int
			if err := e.db.QueryRow(`SELECT count(*) FROM `+table+` WHERE workspace_id = ? AND `+cond, e.session.WorkspaceID).Scan(&liveRows); err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			if err := e.db.QueryRow(`SELECT count(*) FROM `+table+` WHERE workspace_id = ?`, e.session.WorkspaceID).Scan(&allRows); err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			if liveRows != 0 {
				t.Errorf("%s has %d live rows, want 0", table, liveRows)
			}
			if allRows == 0 {
				t.Errorf("%s has no tombstones", table)
			}
		}
	})

	t.Run("lists are empty", func(t *testing.T) {
		projects, err := e.projects.List(e.ctx, e.session.WorkspaceID)
		if err != nil || len(projects) != 0 {
			t.Errorf("projects = %d (err %v), want 0", len(projects), err)
		}
		live, err := e.resourceEnv.svc.List(e.ctx, e.session)
		if err != nil || len(live) != 0 {
			t.Errorf("resources = %d (err %v), want 0", len(live), err)
		}
		trash, err := e.resourceEnv.svc.Trash(e.ctx, e.session)
		if err != nil || len(trash) != 0 {
			t.Errorf("trash = %d (err %v), want 0", len(trash), err)
		}
		if got := e.actions(t, EventQuery{}); len(got) != 0 {
			t.Errorf("events = %v, want none", got)
		}
	})

	t.Run("settings and names are back to defaults", func(t *testing.T) {
		s, err := e.settings.Get(e.ctx, e.session)
		if err != nil || s != defaultUserSettings() {
			t.Errorf("settings = %+v (err %v), want defaults", s, err)
		}
		p, err := e.workspace.Get(e.ctx, e.session)
		if err != nil || p.DisplayName != "tester" || p.WorkspaceName != personalWorkspaceName {
			t.Errorf("profile = %+v (err %v), want tester / %s", p, err, personalWorkspaceName)
		}
	})

	t.Run("files are gone from the disk", func(t *testing.T) {
		wantExists(t, e.disk(), false)
	})

	t.Run("the workspace can be used again", func(t *testing.T) {
		if _, err := e.projects.Create(e.ctx, e.session.WorkspaceID, CreateProjectInput{Name: "Kept"}); err != nil {
			t.Errorf("Create project error = %v", err)
		}
		f := e.upload(t, "", "spec.md", "again")
		if got := e.read(t, f.ID); got != "again" {
			t.Errorf("content = %q, want %q", got, "again")
		}
	})

	t.Run("a second reset succeeds", func(t *testing.T) {
		if err := e.svc.Reset(e.ctx, e.session); err != nil {
			t.Errorf("Reset() error = %v", err)
		}
	})
}

func TestExportAfterReset(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	if err := e.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	doc, files := e.export(t)

	for _, table := range []string{"projects", "tasks", "resources", "resource_events", "task_comments"} {
		if n := len(doc.Tables[table]); n != 0 {
			t.Errorf("%s = %d rows, want 0", table, n)
		}
	}
	if n := len(doc.Tables["memberships"]); n != 1 {
		t.Errorf("memberships = %d rows, want the owner", n)
	}
	if len(files) != 0 {
		t.Errorf("archive files = %v, want none", files)
	}
}
