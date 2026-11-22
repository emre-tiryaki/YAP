package service

import (
	"bytes"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

type fakeAIStopper struct{ stops int }

func (f *fakeAIStopper) StopAll() { f.stops++ }

type aiAccountEnv struct {
	accountEnv
	ai      *repository.AIRepository
	keys    *memAIKeys
	stopper *fakeAIStopper
	taskID  string
	conn    model.AIConnection
	chat    model.AIChat
}

func newAIAccountEnv(t *testing.T) aiAccountEnv {
	t.Helper()
	e := aiAccountEnv{accountEnv: newAccountEnv(t), ai: repository.NewAIRepository(), keys: newMemAIKeys(), stopper: &fakeAIStopper{}}
	e.svc.WithAI(e.ai, e.keys, e.stopper)
	e.seed(t)
	if err := e.db.QueryRow(`SELECT id FROM tasks WHERE workspace_id = ? LIMIT 1`, e.session.WorkspaceID).Scan(&e.taskID); err != nil {
		t.Fatal(err)
	}
	ws, now := e.session.WorkspaceID, nowUTC()
	e.conn = model.AIConnection{ID: model.NewID(), WorkspaceID: ws, UserID: e.session.UserID, Name: "OpenAI", Provider: model.AIProviderOpenAICompatible,
		Preset: model.AIPresetOpenAI, BaseURL: "https://api.openai.com/v1", HasKey: true}
	e.keys.keys[e.conn.ID] = "sk-very-secret"
	e.chat = model.AIChat{ID: model.NewID(), WorkspaceID: ws, UserID: e.session.UserID, Title: "Sprint", ConnectionID: e.conn.ID, Model: "gpt"}
	user := model.AIMessage{ID: model.NewID(), WorkspaceID: ws, ChatID: e.chat.ID, Role: model.AIRoleUser, Content: "hi", Status: model.AIStatusComplete}
	answer := model.AIMessage{ID: model.NewID(), WorkspaceID: ws, ChatID: e.chat.ID, Role: model.AIRoleAssistant, Content: "half", Status: model.AIStatusStreaming,
		ConnectionID: e.conn.ID, Model: "gpt"}
	for _, err := range []error{
		e.ai.InsertConnection(e.ctx, e.db, e.conn, now),
		e.ai.InsertChat(e.ctx, e.db, e.chat, now),
		e.ai.InsertMessage(e.ctx, e.db, user, now),
		e.ai.InsertMessage(e.ctx, e.db, answer, now),
		e.ai.InsertAttachment(e.ctx, e.db, model.AIAttachment{ID: model.NewID(), WorkspaceID: ws, MessageID: user.ID, Position: 0,
			Kind: model.AIAttachTask, TargetID: e.taskID, Label: "T"}, now),
		e.ai.InsertAttachment(e.ctx, e.db, model.AIAttachment{ID: model.NewID(), WorkspaceID: ws, MessageID: user.ID, Position: 1,
			Kind: model.AIAttachRepoFile, TargetID: model.NewID(), Path: "src/main.go", Label: "o/r/src/main.go"}, now),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	return e
}

func (e aiAccountEnv) count(t *testing.T, table, cond string, args ...any) int {
	t.Helper()
	var n int
	if err := e.db.QueryRow(`SELECT count(*) FROM `+table+` WHERE `+cond, args...).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func TestExportHoldsAIDataWithoutKeys(t *testing.T) {
	e := newAIAccountEnv(t)
	var buf bytes.Buffer
	if err := e.svc.Export(e.ctx, e.session, &buf); err != nil {
		t.Fatal(err)
	}
	doc, _ := e.export(t)
	for table, want := range map[string]int{"ai_connections": 1, "ai_chats": 1, "ai_messages": 2, "ai_message_attachments": 2} {
		if got := len(doc.Tables[table]); got != want {
			t.Errorf("%s rows = %d, want %d", table, got, want)
		}
	}
	if strings.Contains(buf.String(), "sk-very-secret") {
		t.Error("export contains an API key")
	}
	// A deleted chat and a deleted connection are left out.
	if err := e.ai.SoftDeleteChat(e.ctx, e.db, e.session.WorkspaceID, e.chat.ID, nowUTC()); err != nil {
		t.Fatal(err)
	}
	doc, _ = e.export(t)
	if len(doc.Tables["ai_chats"]) != 0 || len(doc.Tables["ai_messages"]) != 0 || len(doc.Tables["ai_message_attachments"]) != 0 {
		t.Errorf("deleted chat exported: %v", doc.Tables["ai_chats"])
	}
}

func TestImportRestoresAIDataOnceAndWithoutKeys(t *testing.T) {
	e := newAIAccountEnv(t)
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session, ResetOptions{AIChats: true, AIConnections: true}); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.keys.keys[e.conn.ID]; ok {
		t.Error("reset kept the key")
	}

	res := e.importBytes(t, archive)
	if res.Added.Chats != 1 {
		t.Errorf("Added.Chats = %d, want 1", res.Added.Chats)
	}
	if n := e.count(t, "ai_connections", "workspace_id = ? AND deleted_at IS NULL AND has_key = 0", e.session.WorkspaceID); n != 1 {
		t.Errorf("restored connections without key = %d, want 1", n)
	}
	if n := e.count(t, "ai_messages", "workspace_id = ? AND deleted_at IS NULL", e.session.WorkspaceID); n != 2 {
		t.Errorf("messages = %d, want 2", n)
	}
	if n := e.count(t, "ai_messages", "status = 'interrupted'"); n != 1 {
		t.Errorf("interrupted messages = %d, want the streaming one", n)
	}
	if n := e.count(t, "ai_message_attachments", "workspace_id = ?", e.session.WorkspaceID); n != 2 {
		t.Errorf("attachments = %d, want 2", n)
	}
	if n := e.count(t, "ai_chats", "id = ? AND connection_id = ? AND deleted_at IS NULL", e.chat.ID, e.conn.ID); n != 1 {
		t.Errorf("chat did not keep its id and connection")
	}

	again := e.importBytes(t, archive)
	if again.Added.Chats != 0 {
		t.Errorf("second import Added.Chats = %d, want 0", again.Added.Chats)
	}
	if n := e.count(t, "ai_chats", "workspace_id = ?", e.session.WorkspaceID); n != 1 {
		t.Errorf("chats after second import = %d, want 1", n)
	}
	if n := e.count(t, "ai_message_attachments", "workspace_id = ?", e.session.WorkspaceID); n != 2 {
		t.Errorf("attachments after second import = %d, want 2", n)
	}
}

func TestImportRemapsAIIdsInAnotherWorkspace(t *testing.T) {
	e := newAIAccountEnv(t)
	archive := e.exportBytes(t)
	other := e.otherWorkspace(t)
	if _, err := e.svc.Import(e.ctx, other, bytes.NewReader(archive)); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	var chatID, connID, taskID string
	if err := e.db.QueryRow(`SELECT id, connection_id FROM ai_chats WHERE workspace_id = ?`, other.WorkspaceID).Scan(&chatID, &connID); err != nil {
		t.Fatal(err)
	}
	if chatID == e.chat.ID || connID == e.conn.ID || connID == "" {
		t.Errorf("chat %s / connection %s were not remapped", chatID, connID)
	}
	if n := e.count(t, "ai_messages", "workspace_id = ? AND chat_id = ? AND connection_id IS NULL AND role = 'user'", other.WorkspaceID, chatID); n != 1 {
		t.Errorf("user message not under the new chat")
	}
	if err := e.db.QueryRow(`SELECT id FROM tasks WHERE workspace_id = ?`, other.WorkspaceID).Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	if n := e.count(t, "ai_message_attachments", "workspace_id = ? AND kind = 'task' AND target_id = ?", other.WorkspaceID, taskID); n != 1 {
		t.Errorf("task attachment was not pointed at the copied task")
	}
	// Unknown targets (here a repository that is not in the archive) are kept.
	if n := e.count(t, "ai_message_attachments", "workspace_id = ? AND kind = 'repo_file' AND path = 'src/main.go'", other.WorkspaceID); n != 1 {
		t.Errorf("repo file attachment lost")
	}
}

func TestImportRejectsBrokenAIRows(t *testing.T) {
	tests := []struct {
		name   string
		change func(tables map[string]any)
	}{
		{"unknown preset", func(tb map[string]any) { tb["ai_connections"].([]any)[0].(map[string]any)["preset"] = "nope" }},
		{"bad role", func(tb map[string]any) { tb["ai_messages"].([]any)[0].(map[string]any)["role"] = "system" }},
		{"bad status", func(tb map[string]any) { tb["ai_messages"].([]any)[0].(map[string]any)["status"] = "weird" }},
		{"message without chat", func(tb map[string]any) { tb["ai_messages"].([]any)[0].(map[string]any)["chat_id"] = "x" }},
		{"bad attachment kind", func(tb map[string]any) { tb["ai_message_attachments"].([]any)[0].(map[string]any)["kind"] = "x" }},
		{"path without repo file", func(tb map[string]any) { tb["ai_message_attachments"].([]any)[0].(map[string]any)["path"] = "a" }},
		{"bad time", func(tb map[string]any) { tb["ai_chats"].([]any)[0].(map[string]any)["created_at"] = "yesterday" }},
		{"empty title", func(tb map[string]any) { tb["ai_chats"].([]any)[0].(map[string]any)["title"] = " " }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newAIAccountEnv(t)
			archive := rewrite(t, e.exportBytes(t), func(doc map[string]any) { tt.change(doc["tables"].(map[string]any)) })
			if err := e.svc.Reset(e.ctx, e.session, ResetOptions{AIChats: true, AIConnections: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := e.svc.Import(e.ctx, e.session, bytes.NewReader(archive)); err == nil || !strings.Contains(err.Error(), "invalid input") {
				t.Errorf("Import() error = %v, want invalid input", err)
			}
			if n := e.count(t, "ai_chats", "deleted_at IS NULL"); n != 0 {
				t.Errorf("a rejected import left %d chats", n)
			}
		})
	}
}

func TestResetAIOptions(t *testing.T) {
	tests := []struct {
		name                          string
		opts                          ResetOptions
		wantChats, wantConns, wantKey bool
	}{
		{"neither", ResetOptions{}, true, true, true},
		{"chats", ResetOptions{AIChats: true}, false, true, true},
		{"connections", ResetOptions{AIConnections: true}, true, false, false},
		{"both", ResetOptions{AIChats: true, AIConnections: true}, false, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newAIAccountEnv(t)
			if err := e.svc.Reset(e.ctx, e.session, tt.opts); err != nil {
				t.Fatal(err)
			}
			ws := e.session.WorkspaceID
			if got := e.count(t, "ai_chats", "workspace_id = ? AND deleted_at IS NULL", ws) == 1; got != tt.wantChats {
				t.Errorf("chats kept = %v, want %v", got, tt.wantChats)
			}
			if got := e.count(t, "ai_messages", "workspace_id = ? AND deleted_at IS NULL", ws) == 2; got != tt.wantChats {
				t.Errorf("messages kept = %v, want %v", got, tt.wantChats)
			}
			if got := e.count(t, "ai_connections", "workspace_id = ? AND deleted_at IS NULL", ws) == 1; got != tt.wantConns {
				t.Errorf("connections kept = %v, want %v", got, tt.wantConns)
			}
			if _, got := e.keys.keys[e.conn.ID]; got != tt.wantKey {
				t.Errorf("key kept = %v, want %v", got, tt.wantKey)
			}
			if e.stopper.stops != 1 {
				t.Errorf("stops = %d, want 1", e.stopper.stops)
			}
		})
	}
}
