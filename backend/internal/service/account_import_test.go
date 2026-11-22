package service

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func (e accountEnv) exportBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := e.svc.Export(e.ctx, e.session, &buf); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	return buf.Bytes()
}

func (e accountEnv) importBytes(t *testing.T, archive []byte) ImportResult {
	t.Helper()
	res, err := e.svc.Import(e.ctx, e.session, bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	return res
}

// rewrite changes data.json of an archive; the other entries are copied.
func rewrite(t *testing.T, archive []byte, change func(doc map[string]any)) []byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if f.Name == exportDataName {
			var doc map[string]any
			if err := json.Unmarshal(data, &doc); err != nil {
				t.Fatal(err)
			}
			change(doc)
			if data, err = json.Marshal(doc); err != nil {
				t.Fatal(err)
			}
		}
		w, err := zw.Create(f.Name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func tableRows(doc map[string]any, table string) []any {
	rows, _ := doc["tables"].(map[string]any)[table].([]any)
	return rows
}

func (e accountEnv) projectNames(t *testing.T) []string {
	t.Helper()
	projects, err := e.projects.List(e.ctx, e.session.WorkspaceID)
	if err != nil {
		t.Fatalf("List projects error = %v", err)
	}
	names := []string{}
	for _, p := range projects {
		names = append(names, p.Name)
	}
	slices.Sort(names)
	return names
}

func (e accountEnv) resourcePaths(t *testing.T) map[string]model.Resource {
	t.Helper()
	live, err := e.resourceEnv.svc.List(e.ctx, e.session)
	if err != nil {
		t.Fatalf("List resources error = %v", err)
	}
	byID := map[string]model.Resource{}
	for _, r := range live {
		byID[r.ID] = r
	}
	out := map[string]model.Resource{}
	for _, r := range live {
		path := r.Name
		for p := r.ParentID; p != ""; p = byID[p].ParentID {
			path = byID[p].Name + "/" + path
		}
		out[path] = r
	}
	return out
}

func TestImportRestoresAfterReset(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	before, err := e.projects.List(e.ctx, e.session.WorkspaceID)
	if err != nil {
		t.Fatal(err)
	}
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session, ResetOptions{}); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}

	res := e.importBytes(t, archive)

	if !res.Restored {
		t.Error("Restored = false, want true for an empty workspace")
	}
	want := ImportCounts{Projects: 1, Plans: 1, Tasks: 1, Comments: 1, Resources: 3}
	if res.Added != want {
		t.Errorf("Added = %+v, want %+v", res.Added, want)
	}
	after, err := e.projects.List(e.ctx, e.session.WorkspaceID)
	if err != nil || len(after) != 1 || after[0].ID != before[0].ID || after[0].Name != "Kept" {
		t.Errorf("projects = %+v (err %v), want Kept back under its old id %s", after, err, before[0].ID)
	}
	paths := e.resourcePaths(t)
	spec, ok := paths["docs/spec.md"]
	if !ok {
		t.Fatalf("resources = %v, want docs/spec.md", paths)
	}
	if got := e.read(t, spec.ID); got != "# spec" {
		t.Errorf("spec.md = %q, want %q", got, "# spec")
	}
	if !spec.IsText || spec.Size != int64(len("# spec")) {
		t.Errorf("spec.md is_text %v size %d", spec.IsText, spec.Size)
	}
	if _, ok := paths["docs/empty"]; !ok {
		t.Error("empty folder was not restored")
	}
	tasks, err := e.resourceEnv.svc.ResourceTasks(e.ctx, e.session, spec.ID)
	if err != nil || len(tasks) != 1 || tasks[0].Title != "T" {
		t.Errorf("spec.md linked tasks = %+v (err %v), want task T", tasks, err)
	}
	comments, err := e.comments.List(e.ctx, e.session, tasks[0].TaskID)
	if err != nil || len(comments) != 1 || comments[0].Body != "hello" {
		t.Errorf("comments = %+v (err %v), want hello", comments, err)
	}
	if got := e.actions(t, EventQuery{ResourceID: spec.ID}); !slices.Equal(got, []model.ResourceAction{model.ResourceLinked, model.ResourceUploaded}) {
		t.Errorf("spec.md history = %v, want [linked uploaded]", got)
	}

	s, err := e.settings.Get(e.ctx, e.session)
	if err != nil || s.AutosaveEnabled || s.AutosaveDelayMs != 5000 {
		t.Errorf("settings = %+v (err %v), want the exported ones", s, err)
	}
	p, err := e.workspace.Get(e.ctx, e.session)
	if err != nil || p.DisplayName != "Someone" || p.WorkspaceName != "Work" {
		t.Errorf("profile = %+v (err %v), want Someone / Work", p, err)
	}
}

func TestImportSameArchiveTwiceAddsNothing(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	archive := e.exportBytes(t)

	res := e.importBytes(t, archive)

	if res.Restored {
		t.Error("Restored = true, want false for a workspace with content")
	}
	if res.Added != (ImportCounts{}) {
		t.Errorf("Added = %+v, want nothing", res.Added)
	}
	if res.Existing == 0 {
		t.Error("Existing = 0, want the rows that were recognized")
	}
	if got := e.projectNames(t); !slices.Equal(got, []string{"Kept"}) {
		t.Errorf("projects = %v, want [Kept]", got)
	}
}

func TestImportMergesIntoAnotherWorkspace(t *testing.T) {
	src := newAccountEnv(t)
	src.seed(t)
	archive := src.exportBytes(t)

	dst := newAccountEnv(t)
	if _, err := dst.projects.Create(dst.ctx, dst.session.WorkspaceID, CreateProjectInput{Name: "Other"}); err != nil {
		t.Fatal(err)
	}
	docs := dst.folder(t, "", "Docs") // same folder, other case
	dst.upload(t, docs.ID, "spec.md", "mine")
	on := true
	if _, err := dst.settings.Update(dst.ctx, dst.session, UpdateSettingsInput{AutosaveEnabled: &on}); err != nil {
		t.Fatal(err)
	}

	res := dst.importBytes(t, archive)

	if res.Restored {
		t.Error("Restored = true, want false")
	}
	if res.Renamed != 1 {
		t.Errorf("Renamed = %d, want 1", res.Renamed)
	}
	if got := dst.projectNames(t); !slices.Equal(got, []string{"Kept", "Other"}) {
		t.Errorf("projects = %v, want [Kept Other]", got)
	}
	paths := dst.resourcePaths(t)
	var got []string
	for p := range paths {
		got = append(got, p)
	}
	slices.Sort(got)
	if want := []string{"Docs", "Docs/empty", "Docs/spec (1).md", "Docs/spec.md"}; !slices.Equal(got, want) {
		t.Errorf("resources = %v, want %v", got, want)
	}
	if c := dst.read(t, paths["Docs/spec.md"].ID); c != "mine" {
		t.Errorf("own spec.md = %q, want it untouched", c)
	}
	if c := dst.read(t, paths["Docs/spec (1).md"].ID); c != "# spec" {
		t.Errorf("imported spec = %q, want %q", c, "# spec")
	}
	p, err := dst.workspace.Get(dst.ctx, dst.session)
	if err != nil || p.DisplayName != "tester" || p.WorkspaceName != personalWorkspaceName {
		t.Errorf("profile = %+v (err %v), want it untouched", p, err)
	}
	s, err := dst.settings.Get(dst.ctx, dst.session)
	if err != nil || s != defaultUserSettings() {
		t.Errorf("settings = %+v (err %v), want them untouched", s, err)
	}
}

func TestImportRenamesReaddedPlan(t *testing.T) {
	e := newAccountEnv(t)
	project, err := e.projects.Create(e.ctx, e.session.WorkspaceID, CreateProjectInput{Name: "P"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.projects.CreatePlan(e.ctx, e.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Second"})
	if err != nil {
		t.Fatal(err)
	}
	archive := e.exportBytes(t)
	if err := e.projects.DeletePlan(e.ctx, e.session.WorkspaceID, project.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.projects.CreatePlan(e.ctx, e.session.WorkspaceID, project.ID, CreatePlanInput{Name: "Second"}); err != nil {
		t.Fatal(err)
	}

	res := e.importBytes(t, archive)

	if res.Added.Plans != 1 || res.Added.Projects != 0 {
		t.Errorf("Added = %+v, want only the deleted plan back", res.Added)
	}
	plans, err := e.projects.Plans(e.ctx, e.session.WorkspaceID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, p := range plans {
		names = append(names, p.Name)
	}
	slices.Sort(names)
	if want := []string{"Main", "Second", "Second (2)"}; !slices.Equal(names, want) {
		t.Errorf("plans = %v, want %v", names, want)
	}
}

func TestImportRejectsBadArchives(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	good := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session, ResetOptions{}); err != nil {
		t.Fatal(err)
	}
	first := func(doc map[string]any, table string) map[string]any {
		return tableRows(doc, table)[0].(map[string]any)
	}

	tests := []struct {
		name    string
		archive []byte
	}{
		{"not a zip", []byte("hello")},
		{"no data.json", func() []byte {
			var buf bytes.Buffer
			zw := zip.NewWriter(&buf)
			_, _ = zw.Create("files/x.txt")
			_ = zw.Close()
			return buf.Bytes()
		}()},
		{"other format", rewrite(t, good, func(doc map[string]any) { doc["format"] = "something" })},
		{"newer version", rewrite(t, good, func(doc map[string]any) { doc["version"] = 99 })},
		{"teams", rewrite(t, good, func(doc map[string]any) {
			doc["tables"].(map[string]any)["teams"] = []any{map[string]any{"id": "x"}}
		})},
		{"task in an unknown plan", rewrite(t, good, func(doc map[string]any) { first(doc, "tasks")["plan_id"] = "nope" })},
		{"bad status color", rewrite(t, good, func(doc map[string]any) { first(doc, "statuses")["color"] = "red" })},
		{"bad time", rewrite(t, good, func(doc map[string]any) { first(doc, "projects")["created_at"] = "yesterday" })},
		{"reserved name", rewrite(t, good, func(doc map[string]any) {
			for _, r := range tableRows(doc, "resources") {
				if row := r.(map[string]any); row["parent_id"] == nil {
					row["name"] = ".yap-trash"
				}
			}
		})},
		{"folder cycle", rewrite(t, good, func(doc map[string]any) {
			for _, r := range tableRows(doc, "resources") {
				if row := r.(map[string]any); row["name"] == "docs" {
					row["parent_id"] = row["id"]
				}
			}
		})},
		{"folder in an unknown folder", rewrite(t, good, func(doc map[string]any) {
			for _, r := range tableRows(doc, "resources") {
				if row := r.(map[string]any); row["name"] == "docs" {
					row["parent_id"] = "nope"
				}
			}
		})},
		{"plan without done status", rewrite(t, good, func(doc map[string]any) {
			for _, s := range tableRows(doc, "statuses") {
				s.(map[string]any)["category"] = "todo"
			}
		})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.svc.Import(e.ctx, e.session, bytes.NewReader(tt.archive))
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Import() error = %v, want ErrInvalidInput", err)
			}
			if got := e.projectNames(t); len(got) != 0 {
				t.Errorf("projects = %v, want none", got)
			}
			if got := e.resourcePaths(t); len(got) != 0 {
				t.Errorf("resources = %v, want none", got)
			}
			wantExists(t, e.disk("docs"), false)
		})
	}
}

func TestImportSkipsFilesMissingFromTheArchive(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session, ResetOptions{}); err != nil {
		t.Fatal(err)
	}
	// Drop the file entry, keep its row.
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "spec.md") {
			continue
		}
		if err := zw.Copy(f); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	res := e.importBytes(t, out.Bytes())

	if !slices.Equal(res.MissingFiles, []string{"docs/spec.md"}) {
		t.Errorf("MissingFiles = %v, want [docs/spec.md]", res.MissingFiles)
	}
	if res.Added.Tasks != 1 || res.Added.Resources != 2 {
		t.Errorf("Added = %+v, want the task and both folders", res.Added)
	}
}

// failingStore fails to place one file, after the folders were made.
type failingStore struct {
	AccountStore
	name string
}

func (f failingStore) Place(staged model.StagedFile, path []string) error {
	if path[len(path)-1] == f.name {
		return errors.New("disk full")
	}
	return f.AccountStore.Place(staged, path)
}

func TestImportRollsBackOnFailure(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session, ResetOptions{}); err != nil {
		t.Fatal(err)
	}
	e.svc.store = failingStore{AccountStore: e.svc.store, name: "spec.md"}

	if _, err := e.svc.Import(e.ctx, e.session, bytes.NewReader(archive)); err == nil {
		t.Fatal("Import() error = nil, want the disk error")
	}

	if got := e.projectNames(t); len(got) != 0 {
		t.Errorf("projects = %v, want none", got)
	}
	if got := e.resourcePaths(t); len(got) != 0 {
		t.Errorf("resources = %v, want none", got)
	}
	wantExists(t, e.disk("docs"), false)
}

func TestImportAgainAfterRestoreAddsNothing(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session, ResetOptions{}); err != nil {
		t.Fatal(err)
	}
	e.importBytes(t, archive)

	res := e.importBytes(t, archive)

	if res.Added != (ImportCounts{}) || res.Renamed != 0 {
		t.Errorf("second import added %+v, renamed %d, want nothing", res.Added, res.Renamed)
	}
	if got := e.projectNames(t); !slices.Equal(got, []string{"Kept"}) {
		t.Errorf("projects = %v, want [Kept]", got)
	}
	if got := len(e.resourcePaths(t)); got != 3 {
		t.Errorf("resources = %d, want 3", got)
	}
}

// otherWorkspace adds a second personal workspace with its own owner to the
// same database, so ids of the first one are taken there.
func (e accountEnv) otherWorkspace(t *testing.T) model.Session {
	t.Helper()
	s := model.Session{UserID: model.NewID(), WorkspaceID: model.NewID(), MembershipID: model.NewID()}
	now := nowUTC()
	for _, q := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO users (id, display_name, created_at, updated_at) VALUES (?, 'other', ?, ?)`, []any{s.UserID, now, now}},
		{`INSERT INTO workspaces (id, kind, name, created_at, updated_at) VALUES (?, 'personal', 'Other', ?, ?)`, []any{s.WorkspaceID, now, now}},
		{`INSERT INTO memberships (id, workspace_id, user_id, role, joined_at, updated_at) VALUES (?, ?, ?, 'owner', ?, ?)`,
			[]any{s.MembershipID, s.WorkspaceID, s.UserID, now, now}},
	} {
		if _, err := e.db.Exec(q.query, q.args...); err != nil {
			t.Fatalf("create other workspace: %v", err)
		}
	}
	return s
}

func TestImportGivesNewIdsWhenTakenElsewhere(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	archive := e.exportBytes(t)
	other := e.otherWorkspace(t)

	res, err := e.svc.Import(e.ctx, other, bytes.NewReader(archive))
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}

	if !res.Restored || res.Added.Projects != 1 || res.Added.Tasks != 1 || res.Added.Resources != 3 {
		t.Errorf("result = %+v, want a full copy", res)
	}
	mine, err := e.projects.List(e.ctx, e.session.WorkspaceID)
	if err != nil || len(mine) != 1 {
		t.Fatalf("own projects = %+v (err %v), want them untouched", mine, err)
	}
	theirs, err := e.projects.List(e.ctx, other.WorkspaceID)
	if err != nil || len(theirs) != 1 || theirs[0].ID == mine[0].ID || theirs[0].Name != "Kept" {
		t.Errorf("copied projects = %+v (err %v), want Kept under a new id", theirs, err)
	}
	live, err := e.resourceEnv.svc.List(e.ctx, other)
	if err != nil || len(live) != 3 {
		t.Errorf("copied resources = %d (err %v), want 3", len(live), err)
	}
	wantExists(t, filepath.Join(e.root, other.WorkspaceID, "docs", "spec.md"), true)
	wantExists(t, e.disk("docs", "spec.md"), true)
}

func TestImportCopiesTrashedItemsBack(t *testing.T) {
	e := newAccountEnv(t)
	f := e.upload(t, "", "notes.txt", "keep me")
	archive := e.exportBytes(t)
	if err := e.resourceEnv.svc.MoveToTrash(e.ctx, e.session, f.ID); err != nil {
		t.Fatal(err)
	}

	res := e.importBytes(t, archive)

	if res.Added.Resources != 1 {
		t.Errorf("Added = %+v, want the file back", res.Added)
	}
	paths := e.resourcePaths(t)
	back, ok := paths["notes.txt"]
	if !ok || back.ID == f.ID {
		t.Fatalf("resources = %v, want notes.txt as a new item", paths)
	}
	if got := e.read(t, back.ID); got != "keep me" {
		t.Errorf("content = %q, want %q", got, "keep me")
	}
	if trash, err := e.resourceEnv.svc.Trash(e.ctx, e.session); err != nil || len(trash) != 1 {
		t.Errorf("trash = %d items (err %v), want the original still there", len(trash), err)
	}
}
