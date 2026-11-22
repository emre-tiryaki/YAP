package service

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestTruncateUTF8(t *testing.T) {
	tests := []struct {
		name string
		in   string
		n    int
		want string
	}{
		{"fits", "abc", 3, "abc"},
		{"ascii cut", "abcdef", 4, "abcd"},
		{"zero", "abc", 0, ""},
		{"negative", "abc", -1, ""},
		{"empty", "", 5, ""},
		{"never splits a rune", "aşb", 2, "a"},
		{"keeps whole rune", "aşb", 3, "aş"},
		{"three byte rune", "€€", 5, "€"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateUTF8(tt.in, tt.n)
			if got != tt.want {
				t.Errorf("truncateUTF8(%q, %d) = %q, want %q", tt.in, tt.n, got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("result %q is not valid UTF-8", got)
			}
		})
	}
}

func TestAIStopAllStopsRunningGeneration(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.gate = make(chan struct{})
	e.llm.deltas = []string{"part"}
	res := e.send(t, conn.ID, "go")

	e.chats.StopAll()
	if got := e.chats.Status(); got != "" {
		t.Fatalf("Status() = %q after StopAll", got)
	}
	if m := e.detail(t, res.Chat.ID).Messages[1]; m.Status != model.AIStatusStopped || m.Content != "part" {
		t.Errorf("assistant = %+v", m.AIMessage)
	}
	// Nothing running: a no-op that must not block.
	e.chats.StopAll()
}

func TestRepositoryRunRecoversAndStopsWithContext(t *testing.T) {
	e := newRepoEnv(t)
	r := e.addHello(t, "")
	repo := e.get(t, r.ID)
	repo.State = model.CheckoutFetching
	if err := repository.NewRepoRepository().SaveCheckout(e.ctx, e.db, repo, e.repos.nowString()); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		e.repos.Run(ctx, e.session)
		close(done)
	}()

	// Run recovers the interrupted fetch before its first tick.
	deadline := time.Now().Add(5 * time.Second)
	for e.get(t, r.ID).State == model.CheckoutFetching {
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("Run did not recover the interrupted fetch")
		}
		time.Sleep(2 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return after its context ended")
	}
}

func TestPurgeRemovesTrashedFilesAndLinks(t *testing.T) {
	e := newResourceEnv(t)
	folder, err := e.svc.CreateFolder(e.ctx, e.session, "", "docs")
	if err != nil {
		t.Fatal(err)
	}
	file, err := e.svc.CreateFile(e.ctx, e.session, folder.ID, "a.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.svc.MoveToTrash(e.ctx, e.session, folder.ID); err != nil {
		t.Fatal(err)
	}
	trashDir := filepath.Join(e.root, e.session.WorkspaceID, trashDirName, folder.ID)
	if _, err := os.Stat(trashDir); err != nil {
		t.Fatalf("trashed folder missing on disk: %v", err)
	}

	if err := e.svc.Purge(e.ctx, e.session, folder.ID); err != nil {
		t.Fatalf("Purge() error = %v", err)
	}
	if _, err := os.Stat(trashDir); !os.IsNotExist(err) {
		t.Errorf("purged folder still on disk: %v", err)
	}
	for _, id := range []string{folder.ID, file.ID} {
		if _, err := e.svc.Get(e.ctx, e.session, id); err == nil {
			t.Errorf("Get(%s) after purge succeeded", id)
		}
	}
	if trash, err := e.svc.Trash(e.ctx, e.session); err != nil || len(trash) != 0 {
		t.Errorf("Trash() = %+v, %v", trash, err)
	}
}

func TestEmptyTrashPurgesEveryRoot(t *testing.T) {
	e := newResourceEnv(t)
	var names []string
	for _, n := range []string{"one.txt", "two.txt"} {
		f, err := e.svc.CreateFile(e.ctx, e.session, "", n)
		if err != nil {
			t.Fatal(err)
		}
		if err := e.svc.MoveToTrash(e.ctx, e.session, f.ID); err != nil {
			t.Fatal(err)
		}
		names = append(names, f.ID)
	}
	if err := e.svc.EmptyTrash(e.ctx, e.session); err != nil {
		t.Fatalf("EmptyTrash() error = %v", err)
	}
	if trash, _ := e.svc.Trash(e.ctx, e.session); len(trash) != 0 {
		t.Fatalf("trash after EmptyTrash = %+v", trash)
	}
	entries, err := os.ReadDir(filepath.Join(e.root, e.session.WorkspaceID, trashDirName))
	if err == nil && len(entries) != 0 {
		t.Errorf("trash dir still holds %d entries for %s", len(entries), strings.Join(names, ","))
	}
}

func TestImportRejectsBadLabelIconAndActions(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	e.labelTask(t, "bug")
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
		{"bad label color", rewrite(t, good, func(doc map[string]any) { first(doc, "labels")["color"] = "plaid" })},
		{"blank label name", rewrite(t, good, func(doc map[string]any) { first(doc, "labels")["name"] = "  " })},
		{"bad project icon", rewrite(t, good, func(doc map[string]any) { first(doc, "projects")["icon"] = "abc" })},
		{"bad task event action", rewrite(t, good, func(doc map[string]any) { first(doc, "task_events")["action"] = "exploded" })},
		{"bad resource event action", rewrite(t, good, func(doc map[string]any) { first(doc, "resource_events")["action"] = "exploded" })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := e.svc.Import(e.ctx, e.session, bytes.NewReader(tt.archive)); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Import() error = %v, want ErrInvalidInput", err)
			}
			if got := e.projectNames(t); len(got) != 0 {
				t.Errorf("projects = %v, want none after a rejected import", got)
			}
		})
	}
}

func TestRelationsNotFoundAndRemove(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	var projectID, taskID string
	if err := e.db.QueryRow(`SELECT project_id, id FROM tasks WHERE deleted_at IS NULL`).Scan(&projectID, &taskID); err != nil {
		t.Fatal(err)
	}
	other, err := e.tasks.Create(e.ctx, e.session, projectID, CreateTaskInput{Title: "U"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := e.tasks.Relations(e.ctx, e.session, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Relations(unknown) error = %v", err)
	}
	if err := e.tasks.AddRelation(e.ctx, e.session, "nope", other.ID, model.RelationBlocks); !errors.Is(err, ErrNotFound) {
		t.Errorf("AddRelation(unknown task) error = %v", err)
	}
	if err := e.tasks.RemoveRelation(e.ctx, e.session, "nope", other.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("RemoveRelation(unknown task) error = %v", err)
	}
	if err := e.tasks.RemoveRelation(e.ctx, e.session, taskID, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("RemoveRelation(unknown related) error = %v", err)
	}
	if err := e.tasks.RemoveRelation(e.ctx, e.session, taskID, other.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("RemoveRelation(no relation) error = %v", err)
	}

	if err := e.tasks.AddRelation(e.ctx, e.session, taskID, other.ID, model.RelationBlocks); err != nil {
		t.Fatal(err)
	}
	if got, err := e.tasks.Relations(e.ctx, e.session, taskID); err != nil || len(got) != 1 {
		t.Fatalf("Relations() = %+v, %v", got, err)
	}
	// Either side can remove it.
	if err := e.tasks.RemoveRelation(e.ctx, e.session, other.ID, taskID); err != nil {
		t.Fatalf("RemoveRelation() error = %v", err)
	}
	if got, _ := e.tasks.Relations(e.ctx, e.session, taskID); len(got) != 0 {
		t.Errorf("Relations() after remove = %+v", got)
	}
	if err := e.tasks.AddRelation(e.ctx, e.session, taskID, other.ID, model.RelationRelates); err != nil {
		t.Errorf("re-adding a removed relation error = %v", err)
	}
}
