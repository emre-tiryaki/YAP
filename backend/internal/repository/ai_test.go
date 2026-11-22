package repository

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func seedAIConnection(t *testing.T, db *sql.DB, s workspaceSeed, name string) model.AIConnection {
	t.Helper()
	c := model.AIConnection{
		ID: model.NewID(), WorkspaceID: s.WorkspaceID, UserID: s.UserID, Name: name,
		Provider: model.AIProviderOpenAICompatible, Preset: model.AIPresetOllama, BaseURL: "http://localhost:11434/v1",
	}
	if err := NewAIRepository().InsertConnection(context.Background(), db, c, baseNow); err != nil {
		t.Fatal(err)
	}
	return c
}

func seedAIChat(t *testing.T, db *sql.DB, s workspaceSeed, title, connID, now string) model.AIChat {
	t.Helper()
	c := model.AIChat{ID: model.NewID(), WorkspaceID: s.WorkspaceID, UserID: s.UserID, Title: title, ConnectionID: connID, Model: "m"}
	if err := NewAIRepository().InsertChat(context.Background(), db, c, now); err != nil {
		t.Fatal(err)
	}
	return c
}

func seedAIMessage(t *testing.T, db *sql.DB, chat model.AIChat, role model.AIMessageRole, content string, status model.AIMessageStatus, now string) model.AIMessage {
	t.Helper()
	m := model.AIMessage{ID: model.NewID(), WorkspaceID: chat.WorkspaceID, ChatID: chat.ID, Role: role, Content: content, Status: status}
	if err := NewAIRepository().InsertMessage(context.Background(), db, m, now); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestAIConnections(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	s := seedWorkspace(t, db, "a")
	other := seedWorkspace(t, db, "b")
	repo := NewAIRepository()

	c := seedAIConnection(t, db, s, "Local")
	seedAIConnection(t, db, other, "Other")

	list, err := repo.ListConnections(ctx, db, s.WorkspaceID, s.UserID)
	if err != nil || len(list) != 1 || list[0].ID != c.ID {
		t.Fatalf("ListConnections() = %v, %v", list, err)
	}
	if err := repo.UpdateConnection(ctx, db, s.WorkspaceID, c.ID, "Renamed", "http://x/v1", true, "2026-02-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetConnection(ctx, db, s.WorkspaceID, s.UserID, c.ID)
	if err != nil || got.Name != "Renamed" || !got.HasKey || got.BaseURL != "http://x/v1" {
		t.Fatalf("GetConnection() = %+v, %v", got, err)
	}
	if _, err := repo.GetConnection(ctx, db, other.WorkspaceID, other.UserID, c.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetConnection() from another workspace error = %v, want ErrNoRows", err)
	}
	if err := repo.SoftDeleteConnection(ctx, db, s.WorkspaceID, c.ID, baseNow); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetConnection(ctx, db, s.WorkspaceID, s.UserID, c.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("deleted connection error = %v, want ErrNoRows", err)
	}
	live, err := repo.LiveConnectionIDs(ctx, db, s.WorkspaceID, []string{c.ID})
	if err != nil || len(live) != 0 {
		t.Errorf("LiveConnectionIDs() = %v, %v", live, err)
	}
}

func TestAIChatsListOrderSearchAndDelete(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	s := seedWorkspace(t, db, "a")
	repo := NewAIRepository()

	old := seedAIChat(t, db, s, "Eski plan", "", "2026-01-01T00:00:00Z")
	newer := seedAIChat(t, db, s, "ÖDEV özeti", "", "2026-01-03T00:00:00Z")
	mid := seedAIChat(t, db, s, "Sprint", "", "2026-01-02T00:00:00Z")

	tests := []struct {
		name   string
		needle string
		want   []string
	}{
		{"all by recency", "", []string{newer.ID, mid.ID, old.ID}},
		{"turkish folding", database.Fold("ödev"), []string{newer.ID}},
		{"substring", "pri", []string{mid.ID}},
		{"no hit", "zzz", []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			list, err := repo.ListChats(ctx, db, s.WorkspaceID, s.UserID, tt.needle)
			if err != nil {
				t.Fatal(err)
			}
			ids := []string{}
			for _, c := range list {
				ids = append(ids, c.ID)
			}
			if !reflect.DeepEqual(ids, tt.want) {
				t.Errorf("ids = %v, want %v", ids, tt.want)
			}
		})
	}

	m := seedAIMessage(t, db, mid, model.AIRoleUser, "hi", model.AIStatusComplete, baseNow)
	if err := repo.SoftDeleteChat(ctx, db, s.WorkspaceID, mid.ID, baseNow); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetChat(ctx, db, s.WorkspaceID, s.UserID, mid.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("GetChat() after delete error = %v", err)
	}
	msgs, err := repo.ListMessages(ctx, db, s.WorkspaceID, mid.ID)
	if err != nil || len(msgs) != 0 {
		t.Errorf("ListMessages() after chat delete = %v, %v (message %s)", msgs, err, m.ID)
	}
}

func TestAIMessagesAndAttachments(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	s := seedWorkspace(t, db, "a")
	repo := NewAIRepository()
	chat := seedAIChat(t, db, s, "c", "", baseNow)

	u := seedAIMessage(t, db, chat, model.AIRoleUser, "q", model.AIStatusComplete, "2026-01-01T00:00:01Z")
	a := seedAIMessage(t, db, chat, model.AIRoleAssistant, "", model.AIStatusStreaming, "2026-01-01T00:00:02Z")

	for i, att := range []model.AIAttachment{
		{Kind: model.AIAttachRepoFile, TargetID: model.NewID(), Path: "src/a.go", Label: "o/r/src/a.go"},
		{Kind: model.AIAttachTask, TargetID: model.NewID(), Label: "ABC-1 t"},
	} {
		att.ID, att.WorkspaceID, att.MessageID, att.Position = model.NewID(), s.WorkspaceID, u.ID, i
		if err := repo.InsertAttachment(ctx, db, att, baseNow); err != nil {
			t.Fatal(err)
		}
	}
	bad := model.AIAttachment{ID: model.NewID(), WorkspaceID: s.WorkspaceID, MessageID: u.ID, Kind: model.AIAttachTask, TargetID: "x", Path: "p", Label: "l"}
	if err := repo.InsertAttachment(ctx, db, bad, baseNow); err == nil {
		t.Error("InsertAttachment() with path on a task: expected the CHECK to fail")
	}

	atts, err := repo.ListAttachments(ctx, db, s.WorkspaceID, []string{u.ID, a.ID})
	if err != nil || len(atts[u.ID]) != 2 || atts[u.ID][0].Kind != model.AIAttachRepoFile || atts[u.ID][1].Position != 1 {
		t.Fatalf("ListAttachments() = %+v, %v", atts, err)
	}
	if err := repo.SetAttachmentStatuses(ctx, db, s.WorkspaceID, map[string]model.AIAttachmentStatus{atts[u.ID][0].ID: model.AIAttachTruncated}, baseNow); err != nil {
		t.Fatal(err)
	}

	if err := repo.UpdateMessageContent(ctx, db, s.WorkspaceID, a.ID, "partial", baseNow); err != nil {
		t.Fatal(err)
	}
	n, err := repo.MarkStreamingInterrupted(ctx, db, baseNow)
	if err != nil || n != 1 {
		t.Fatalf("MarkStreamingInterrupted() = %d, %v", n, err)
	}
	msgs, _ := repo.ListMessages(ctx, db, s.WorkspaceID, chat.ID)
	if len(msgs) != 2 || msgs[0].ID != u.ID || msgs[1].Status != model.AIStatusInterrupted || msgs[1].Content != "partial" {
		t.Errorf("ListMessages() = %+v", msgs)
	}
	if err := repo.FinishMessage(ctx, db, s.WorkspaceID, a.ID, "done", model.AIStatusError, "ai_unreachable", baseNow); err != nil {
		t.Fatal(err)
	}
	msgs, _ = repo.ListMessages(ctx, db, s.WorkspaceID, chat.ID)
	if msgs[1].ErrorCode != "ai_unreachable" || msgs[1].Content != "done" {
		t.Errorf("FinishMessage() row = %+v", msgs[1])
	}

	if err := repo.SoftDeleteMessage(ctx, db, s.WorkspaceID, u.ID, baseNow); err != nil {
		t.Fatal(err)
	}
	atts, _ = repo.ListAttachments(ctx, db, s.WorkspaceID, []string{u.ID})
	if len(atts) != 0 {
		t.Errorf("attachments of a deleted message = %v", atts)
	}
}

func TestAILiveHelpers(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	s := seedWorkspace(t, db, "a")
	repo := NewAIRepository()

	proj := seedProject(t, db, s.WorkspaceID, "P", baseNow)
	plan := seedPlan(t, db, s.WorkspaceID, proj.ID, "Q")
	st := seedStatuses(t, db, s.WorkspaceID, plan.ID)
	task := seedTask(t, db, s, plan, st[model.StatusToDo], "t", "a1", nil)
	gone := seedTask(t, db, s, plan, st[model.StatusToDo], "g", "a2", nil)
	mustExec(t, db, `UPDATE tasks SET deleted_at = ? WHERE id = ?`, baseNow, gone.ID)

	tasks, err := repo.LiveTaskPlacements(ctx, db, s.WorkspaceID, []string{task.ID, gone.ID, "nope"})
	if err != nil || len(tasks) != 1 || tasks[task.ID] != (model.AITaskPlacement{ProjectID: proj.ID, PlanID: plan.ID}) {
		t.Errorf("LiveTaskPlacements() = %v, %v", tasks, err)
	}
	plans, err := repo.LivePlanProjects(ctx, db, s.WorkspaceID, []string{plan.ID})
	if err != nil || plans[plan.ID] != proj.ID {
		t.Errorf("LivePlanProjects() = %v, %v", plans, err)
	}
	projs, err := repo.LiveProjectIDs(ctx, db, s.WorkspaceID, []string{proj.ID, "x"})
	if err != nil || !projs[proj.ID] || projs["x"] {
		t.Errorf("LiveProjectIDs() = %v, %v", projs, err)
	}
	res, err := repo.LiveResourceIDs(ctx, db, s.WorkspaceID, nil)
	if err != nil || len(res) != 0 {
		t.Errorf("LiveResourceIDs(nil) = %v, %v", res, err)
	}

	mustExec(t, db, `UPDATE projects SET deleted_at = ? WHERE id = ?`, baseNow, proj.ID)
	tasks, _ = repo.LiveTaskPlacements(ctx, db, s.WorkspaceID, []string{task.ID})
	if len(tasks) != 0 {
		t.Errorf("task of a deleted project still live: %v", tasks)
	}
}

func TestAIResetHelpers(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	s := seedWorkspace(t, db, "a")
	repo := NewAIRepository()
	c := seedAIConnection(t, db, s, "x")
	chat := seedAIChat(t, db, s, "c", c.ID, baseNow)
	seedAIMessage(t, db, chat, model.AIRoleUser, "q", model.AIStatusComplete, baseNow)

	if err := repo.SoftDeleteAllChats(ctx, db, s.WorkspaceID, s.UserID, baseNow); err != nil {
		t.Fatal(err)
	}
	if list, _ := repo.ListChats(ctx, db, s.WorkspaceID, s.UserID, ""); len(list) != 0 {
		t.Errorf("chats after reset = %v", list)
	}
	ids, err := repo.SoftDeleteAllConnections(ctx, db, s.WorkspaceID, s.UserID, baseNow)
	if err != nil || !reflect.DeepEqual(ids, []string{c.ID}) {
		t.Errorf("SoftDeleteAllConnections() = %v, %v", ids, err)
	}
}
