package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func (e aiEnv) send(t *testing.T, connID, content string) AISendResult {
	t.Helper()
	res, err := e.chats.Create(e.ctx, e.session, SendAIInput{ConnectionID: connID, Model: "m1", Content: content})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	return res
}

// waitIdle blocks until the running generation has written its final state.
// It waits on the service's own work group instead of polling, and fails
// rather than hangs when a test forgot to open the model's gate.
func (e aiEnv) waitIdle(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		e.chats.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("generation did not finish")
	}
	if got := e.chats.Status(); got != "" {
		t.Fatalf("Status() = %q after the generation finished", got)
	}
}

func (e aiEnv) detail(t *testing.T, chatID string) AIChatDetail {
	t.Helper()
	d, err := e.chats.Get(e.ctx, e.session, chatID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	return d
}

func TestAIChatCreateCompletes(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.deltas = []string{"Hel", "lo"}

	long := strings.Repeat("ş", 70)
	res := e.send(t, conn.ID, "  \n"+long+"  ")
	if want := strings.Repeat("ş", 60) + "…"; res.Chat.Title != want {
		t.Errorf("title = %q, want %q", res.Chat.Title, want)
	}
	if res.AssistantMessage.Status != model.AIStatusStreaming || res.UserMessage.Content != long {
		t.Errorf("result = %+v", res)
	}
	e.waitIdle(t)
	d := e.detail(t, res.Chat.ID)
	if len(d.Messages) != 2 || d.Messages[1].Content != "Hello" || d.Messages[1].Status != model.AIStatusComplete ||
		d.Messages[1].ConnectionID != conn.ID || d.Messages[1].Model != "m1" || d.Chat.Generating {
		t.Errorf("detail = %+v", d)
	}
	if req := e.llm.lastRequest(); req.System != aiSystemPrompt || len(req.Messages) != 1 || req.Model != "m1" {
		t.Errorf("request = %+v", req)
	}

	// Second message: history goes along.
	if _, err := e.chats.Send(e.ctx, e.session, res.Chat.ID, SendAIInput{ConnectionID: conn.ID, Model: "m2", Content: "more"}); err != nil {
		t.Fatal(err)
	}
	e.waitIdle(t)
	if req := e.llm.lastRequest(); len(req.Messages) != 3 || req.Messages[1].Content != "Hello" || req.Model != "m2" {
		t.Errorf("second request = %+v", req)
	}
	if d := e.detail(t, res.Chat.ID); d.Chat.Model != "m2" || len(d.Messages) != 4 {
		t.Errorf("chat model = %q, messages %d", d.Chat.Model, len(d.Messages))
	}
}

func TestAIChatInstructionsGoToSystemPrompt(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.deltas = []string{"ok"}
	text := "Always answer in Turkish."
	if _, err := NewSettingsService(e.db, e.aiSettings()).Update(e.ctx, e.session, UpdateSettingsInput{AIInstructions: &text}); err != nil {
		t.Fatal(err)
	}
	e.send(t, conn.ID, "hi")
	e.waitIdle(t)
	if got := e.llm.lastRequest().System; !strings.Contains(got, text) {
		t.Errorf("system = %q", got)
	}
}

func TestAIChatBusy(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.gate = make(chan struct{})
	e.llm.deltas = []string{"a"}
	res := e.send(t, conn.ID, "first")

	in := SendAIInput{ConnectionID: conn.ID, Model: "m1", Content: "second"}
	if _, err := e.chats.Create(e.ctx, e.session, in); !errors.Is(err, ErrAIBusy) {
		t.Errorf("Create() while busy error = %v", err)
	}
	if _, err := e.chats.Send(e.ctx, e.session, res.Chat.ID, in); !errors.Is(err, ErrAIBusy) {
		t.Errorf("Send() while busy error = %v", err)
	}
	if _, err := e.chats.Regenerate(e.ctx, e.session, res.Chat.ID, RegenerateAIInput{ConnectionID: conn.ID, Model: "m1"}); !errors.Is(err, ErrAIBusy) {
		t.Errorf("Regenerate() while busy error = %v", err)
	}
	if e.chats.Status() != res.Chat.ID {
		t.Errorf("Status() = %q", e.chats.Status())
	}
	chats, err := e.chats.List(e.ctx, e.session, "")
	if err != nil || len(chats) != 1 || !chats[0].Generating {
		t.Errorf("List() = %+v (err %v)", chats, err)
	}
	close(e.llm.gate)
	e.waitIdle(t)
	if _, err := e.chats.Create(e.ctx, e.session, in); err != nil {
		t.Errorf("Create() after finish error = %v", err)
	}
}

func TestAIChatStopKeepsPartialAnswer(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.gate = make(chan struct{})
	e.llm.deltas = []string{"part", "ial"}
	res := e.send(t, conn.ID, "go")

	if err := e.chats.Stop(e.ctx, e.session, res.Chat.ID); err != nil {
		t.Fatal(err)
	}
	if e.chats.Status() != "" {
		t.Error("still generating after Stop")
	}
	d := e.detail(t, res.Chat.ID)
	if m := d.Messages[1]; m.Status != model.AIStatusStopped || m.Content != "partial" {
		t.Errorf("assistant = %+v", m.AIMessage)
	}
	// Idempotent, and unknown chats are not found.
	if err := e.chats.Stop(e.ctx, e.session, res.Chat.ID); err != nil {
		t.Errorf("second Stop() error = %v", err)
	}
	if err := e.chats.Stop(e.ctx, e.session, "nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Stop(unknown) error = %v", err)
	}
}

func TestAIChatShutdownInterrupts(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	ctx, cancel := context.WithCancel(context.Background())
	e.chats.Start(ctx)
	e.llm.gate = make(chan struct{})
	e.llm.deltas = []string{"half"}
	res := e.send(t, conn.ID, "go")
	cancel()
	e.chats.Wait()
	if m := e.detail(t, res.Chat.ID).Messages[1]; m.Status != model.AIStatusInterrupted || m.Content != "half" {
		t.Errorf("assistant = %+v", m.AIMessage)
	}
}

func TestAIChatProviderErrors(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		delta []string
		code  string
	}{
		{"unreachable", ErrAIUnreachable, nil, "ai_unreachable"},
		{"auth", ErrAIAuthFailed, nil, "ai_auth_failed"},
		{"rate limit", ErrAIRateLimited, []string{"x"}, "ai_rate_limited"},
		{"model", ErrAIModelNotFound, nil, "ai_model_not_found"},
		{"rejected", ErrAIRejected, nil, "ai_rejected"},
		{"other", errors.New("boom"), nil, "internal"},
		{"empty answer", nil, nil, "ai_rejected"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newAIEnv(t)
			conn := e.connection(t)
			e.llm.err, e.llm.deltas = tt.err, tt.delta
			res := e.send(t, conn.ID, "go")
			e.waitIdle(t)
			m := e.detail(t, res.Chat.ID).Messages[1]
			if m.Status != model.AIStatusError || m.ErrorCode != tt.code {
				t.Errorf("assistant = %+v, want error %s", m.AIMessage, tt.code)
			}
		})
	}
}

func TestAIChatRegenerate(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.deltas = []string{"one"}
	res := e.send(t, conn.ID, "q")
	e.waitIdle(t)

	e.llm.deltas = []string{"two"}
	if _, err := e.chats.Regenerate(e.ctx, e.session, res.Chat.ID, RegenerateAIInput{ConnectionID: conn.ID, Model: "m9"}); err != nil {
		t.Fatal(err)
	}
	e.waitIdle(t)
	d := e.detail(t, res.Chat.ID)
	if len(d.Messages) != 2 || d.Messages[1].Content != "two" || d.Messages[1].Model != "m9" {
		t.Errorf("messages = %+v", d.Messages)
	}
	if req := e.llm.lastRequest(); len(req.Messages) != 1 {
		t.Errorf("request has %d messages, want only the question", len(req.Messages))
	}

	// A failed answer can be regenerated too; a lone question as well.
	e.llm.err = ErrAIUnreachable
	e.llm.deltas = nil
	if _, err := e.chats.Regenerate(e.ctx, e.session, res.Chat.ID, RegenerateAIInput{ConnectionID: conn.ID, Model: "m1"}); err != nil {
		t.Fatal(err)
	}
	e.waitIdle(t)
	e.llm.err, e.llm.deltas = nil, []string{"three"}
	if _, err := e.chats.Regenerate(e.ctx, e.session, res.Chat.ID, RegenerateAIInput{ConnectionID: conn.ID, Model: "m1"}); err != nil {
		t.Fatal(err)
	}
	e.waitIdle(t)
	if d := e.detail(t, res.Chat.ID); len(d.Messages) != 2 || d.Messages[1].Content != "three" {
		t.Errorf("messages = %+v", d.Messages)
	}
}

func TestAIChatRegenerateRejectsStreamingAndEmpty(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.gate = make(chan struct{})
	e.llm.deltas = []string{"a"}
	res := e.send(t, conn.ID, "q")
	// While streaming the reservation answers first.
	if _, err := e.chats.Regenerate(e.ctx, e.session, res.Chat.ID, RegenerateAIInput{ConnectionID: conn.ID, Model: "m"}); !errors.Is(err, ErrAIBusy) {
		t.Errorf("error = %v, want ErrAIBusy", err)
	}
	// A stale streaming row (no active generation) is refused as well.
	close(e.llm.gate)
	e.waitIdle(t)
	if _, err := e.db.Exec(`UPDATE ai_messages SET status = 'streaming' WHERE role = 'assistant'`); err != nil {
		t.Fatal(err)
	}
	if _, err := e.chats.Regenerate(e.ctx, e.session, res.Chat.ID, RegenerateAIInput{ConnectionID: conn.ID, Model: "m"}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("error = %v, want ErrInvalidInput", err)
	}
	if _, err := e.chats.Regenerate(e.ctx, e.session, "nope", RegenerateAIInput{ConnectionID: conn.ID, Model: "m"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
	if e.chats.Status() != "" {
		t.Error("reservation leaked")
	}
}

func TestAIChatConnectionErrors(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.deltas = []string{"ok"}
	in := SendAIInput{ConnectionID: conn.ID, Model: "m1", Content: "hi"}

	if err := e.conns.Delete(e.ctx, e.session, conn.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.chats.Create(e.ctx, e.session, in); !errors.Is(err, ErrAIConnectionMissing) {
		t.Errorf("deleted connection error = %v", err)
	}
	keyed, err := e.conns.Create(e.ctx, e.session, CreateAIConnectionInput{Name: "K", Preset: model.AIPresetOpenAI, BaseURL: "https://api.openai.com/v1", APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	delete(e.keys.keys, keyed.ID)
	in.ConnectionID = keyed.ID
	if _, err := e.chats.Create(e.ctx, e.session, in); !errors.Is(err, ErrAIKeyMissing) {
		t.Errorf("missing key error = %v", err)
	}
	if e.chats.Status() != "" {
		t.Error("reservation leaked")
	}
	if chats, _ := e.chats.List(e.ctx, e.session, ""); len(chats) != 0 {
		t.Errorf("failed send created chats: %+v", chats)
	}
}

func TestAIChatInputValidation(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	ok := SendAIInput{ConnectionID: conn.ID, Model: "m1", Content: "hi"}
	tests := []struct {
		name   string
		mutate func(*SendAIInput)
	}{
		{"blank", func(in *SendAIInput) { in.Content = "  " }},
		{"too long", func(in *SendAIInput) { in.Content = strings.Repeat("a", 32_001) }},
		{"no model", func(in *SendAIInput) { in.Model = "" }},
		{"long model", func(in *SendAIInput) { in.Model = strings.Repeat("m", 201) }},
		{"no connection", func(in *SendAIInput) { in.ConnectionID = "" }},
		{"too many attachments", func(in *SendAIInput) {
			in.Attachments = make([]AIAttachmentInput, 21)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := ok
			tt.mutate(&in)
			if _, err := e.chats.Create(e.ctx, e.session, in); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("Create() error = %v, want ErrInvalidInput", err)
			}
		})
	}
	if e.chats.Status() != "" {
		t.Error("reservation leaked")
	}
}

func TestAIChatAttachments(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.deltas = []string{"ok"}
	project := e.project(t, "Backend")
	task := e.task(t, project.ID, CreateTaskInput{Title: "Fix login"})
	planID := e.planOf(t, project.ID)
	notes := e.upload(t, "", "notes.md", "# Notes")
	bin := e.upload(t, "", "logo.bin", "\x00\x01")
	folder := e.folder(t, "", "docs")
	trashed := e.upload(t, "", "old.md", "old")
	if err := e.svc.MoveToTrash(e.ctx, e.session, trashed.ID); err != nil {
		t.Fatal(err)
	}
	repo := e.addHello(t, "")

	bad := []struct {
		name string
		att  AIAttachmentInput
		want error
	}{
		{"binary file", AIAttachmentInput{Kind: model.AIAttachResource, TargetID: bin.ID}, ErrInvalidInput},
		{"folder", AIAttachmentInput{Kind: model.AIAttachResource, TargetID: folder.ID}, ErrInvalidInput},
		{"trashed", AIAttachmentInput{Kind: model.AIAttachResource, TargetID: trashed.ID}, ErrNotFound},
		{"unknown task", AIAttachmentInput{Kind: model.AIAttachTask, TargetID: model.NewID()}, ErrNotFound},
		{"wrong kind", AIAttachmentInput{Kind: model.AIAttachPlan, TargetID: task.ID}, ErrNotFound},
		{"unknown kind", AIAttachmentInput{Kind: "nope", TargetID: task.ID}, ErrInvalidInput},
		{"path on task", AIAttachmentInput{Kind: model.AIAttachTask, TargetID: task.ID, Path: "a"}, ErrInvalidInput},
		{"repo file without path", AIAttachmentInput{Kind: model.AIAttachRepoFile, TargetID: repo.ID}, ErrInvalidInput},
		{"repo binary", AIAttachmentInput{Kind: model.AIAttachRepoFile, TargetID: repo.ID, Path: "logo.bin"}, ErrInvalidInput},
		{"repo missing file", AIAttachmentInput{Kind: model.AIAttachRepoFile, TargetID: repo.ID, Path: "nope.go"}, ErrNotFound},
		{"repo traversal", AIAttachmentInput{Kind: model.AIAttachRepoFile, TargetID: repo.ID, Path: "../x"}, ErrInvalidInput},
		{"repo id of a plain file", AIAttachmentInput{Kind: model.AIAttachRepoFile, TargetID: notes.ID, Path: "a"}, ErrNotFound},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			_, err := e.chats.Create(e.ctx, e.session, SendAIInput{ConnectionID: conn.ID, Model: "m1", Content: "hi", Attachments: []AIAttachmentInput{tt.att}})
			if !errors.Is(err, tt.want) {
				t.Errorf("Create() error = %v, want %v", err, tt.want)
			}
		})
	}
	if chats, _ := e.chats.List(e.ctx, e.session, ""); len(chats) != 0 || e.chats.Status() != "" {
		t.Fatalf("a rejected send left state behind: %+v", chats)
	}

	res, err := e.chats.Create(e.ctx, e.session, SendAIInput{ConnectionID: conn.ID, Model: "m1", Content: "summarize", Attachments: []AIAttachmentInput{
		{Kind: model.AIAttachTask, TargetID: task.ID},
		{Kind: model.AIAttachPlan, TargetID: planID},
		{Kind: model.AIAttachProject, TargetID: project.ID},
		{Kind: model.AIAttachResource, TargetID: notes.ID},
		{Kind: model.AIAttachRepoFile, TargetID: repo.ID, Path: "src/main.go"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	e.waitIdle(t)
	atts := e.detail(t, res.Chat.ID).Messages[0].Attachments
	labels := []string{}
	for _, a := range atts {
		if a.Missing || a.LastStatus != model.AIAttachIncluded {
			t.Errorf("attachment %+v", a)
		}
		labels = append(labels, a.Label)
	}
	if len(atts) != 5 || atts[0].Label != project.Identifier+"-1 Fix login" || atts[1].Label != "Backend / "+atts[1].Label[len("Backend / "):] ||
		atts[2].Label != "Backend" || atts[3].Label != "notes.md" || atts[4].Label != "octo/hello/src/main.go" {
		t.Errorf("labels = %v", labels)
	}
	if atts[0].ProjectID != project.ID || atts[0].PlanID != planID || atts[1].ProjectID != project.ID || atts[2].ProjectID != project.ID {
		t.Errorf("links = %+v", atts)
	}
	if content := e.llm.lastRequest().Messages[0].Content; !strings.Contains(content, `kind="task"`) || !strings.HasSuffix(content, "\n\nsummarize") {
		t.Errorf("prompt = %q", content)
	}

	// Deleting a target keeps the chat usable: the chip shows as missing and the next send still works.
	if err := e.tasks.Delete(e.ctx, e.session, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.chats.Send(e.ctx, e.session, res.Chat.ID, SendAIInput{ConnectionID: conn.ID, Model: "m1", Content: "again"}); err != nil {
		t.Fatal(err)
	}
	e.waitIdle(t)
	d := e.detail(t, res.Chat.ID)
	if a := d.Messages[0].Attachments[0]; !a.Missing || a.LastStatus != model.AIAttachMissing {
		t.Errorf("deleted task attachment = %+v", a)
	}
	if content := e.llm.lastRequest().Messages[0].Content; !strings.Contains(content, `kind="task" name="`+atts[0].Label+`" missing="true"/>`) {
		t.Errorf("prompt = %q", content)
	}
}

func TestAIChatRecoverInterrupted(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.deltas = []string{"x"}
	res := e.send(t, conn.ID, "q")
	e.waitIdle(t)
	if _, err := e.db.Exec(`UPDATE ai_messages SET status = 'streaming' WHERE role = 'assistant'`); err != nil {
		t.Fatal(err)
	}
	if err := e.chats.RecoverInterrupted(e.ctx); err != nil {
		t.Fatal(err)
	}
	if m := e.detail(t, res.Chat.ID).Messages[1]; m.Status != model.AIStatusInterrupted {
		t.Errorf("status = %s", m.Status)
	}
}

func TestAIChatDeleteStopsGeneration(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.gate = make(chan struct{})
	e.llm.deltas = []string{"x"}
	res := e.send(t, conn.ID, "q")
	if err := e.chats.Delete(e.ctx, e.session, res.Chat.ID); err != nil {
		t.Fatal(err)
	}
	if e.chats.Status() != "" {
		t.Error("still generating after Delete")
	}
	if _, err := e.chats.Get(e.ctx, e.session, res.Chat.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get() after delete error = %v", err)
	}
	if err := e.chats.Delete(e.ctx, e.session, res.Chat.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("second Delete() error = %v", err)
	}
}

func TestAIChatListSearchAndRename(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.deltas = []string{"ok"}
	a := e.send(t, conn.ID, "ISPARTA planı")
	e.waitIdle(t)
	e.send(t, conn.ID, "Sprint summary")
	e.waitIdle(t)

	list, err := e.chats.List(e.ctx, e.session, "sprint")
	if err != nil || len(list) != 1 || list[0].Title != "Sprint summary" {
		t.Errorf("List(sprint) = %+v (err %v)", list, err)
	}
	if list, _ := e.chats.List(e.ctx, e.session, ""); len(list) != 2 {
		t.Errorf("List() = %d chats", len(list))
	}
	if _, err := e.chats.List(e.ctx, e.session, strings.Repeat("a", 201)); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("long search error = %v", err)
	}

	renamed, err := e.chats.Rename(e.ctx, e.session, a.Chat.ID, "  New name ")
	if err != nil || renamed.Title != "New name" {
		t.Errorf("Rename() = %+v (err %v)", renamed, err)
	}
	for _, title := range []string{" ", strings.Repeat("a", 201)} {
		if _, err := e.chats.Rename(e.ctx, e.session, a.Chat.ID, title); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Rename(%q) error = %v", title, err)
		}
	}
}

func TestAIChatStreamSubscription(t *testing.T) {
	e := newAIEnv(t)
	conn := e.connection(t)
	e.llm.gate = make(chan struct{})
	e.llm.deltas = []string{"ab"}
	res := e.send(t, conn.ID, "q")

	if _, ok := e.chats.Subscribe("other"); ok {
		t.Error("Subscribe(other chat) = true")
	}
	st, ok := e.chats.Subscribe(res.Chat.ID)
	if !ok || st.MessageID() != res.AssistantMessage.ID {
		t.Fatalf("Subscribe() = %v, %q", ok, st.MessageID())
	}
	// The generation runs in its own goroutine: wait for the deltas.
	text, next, changed, done, _ := st.Read(0)
	for next < 2 && !done {
		<-changed
		text, next, changed, done, _ = st.Read(0)
	}
	if text != "ab" || next != 2 || done {
		t.Errorf("Read(0) = %q, %d, %v", text, next, done)
	}
	close(e.llm.gate)
	<-changed
	for {
		text, _, changed, done, status := st.Read(next)
		if done {
			if text != "" || status != model.AIStatusComplete {
				t.Errorf("done read = %q, %s", text, status)
			}
			break
		}
		<-changed
	}
	e.waitIdle(t)
	if _, ok := e.chats.Subscribe(res.Chat.ID); ok {
		t.Error("Subscribe() after finish = true")
	}
}
