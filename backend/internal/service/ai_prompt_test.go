package service

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func attach(kind model.AIAttachmentKind, target, label string) model.AIAttachment {
	return model.AIAttachment{ID: model.NewID(), Kind: kind, TargetID: target, Label: label}
}

func (e aiEnv) text(t *testing.T, a model.AIAttachment, remaining int) (string, model.AIAttachmentStatus, int) {
	t.Helper()
	s, st, used, err := e.prompt.attachmentText(e.ctx, e.session, a, remaining)
	if err != nil {
		t.Fatalf("attachmentText() error = %v", err)
	}
	return s, st, used
}

func TestAIPromptTaskPlanProject(t *testing.T) {
	e := newAIEnv(t)
	project := e.project(t, "Backend")
	due, start := time.Date(2026, 11, 30, 0, 0, 0, 0, time.UTC), time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	task := e.task(t, project.ID, CreateTaskInput{Title: "Fix login", Description: "Users cannot log in.", DueDate: &due, StartDate: &start, Priority: model.PriorityHigh})
	sub, err := e.tasks.CreateSubtask(e.ctx, e.session, task.ID, "Reproduce")
	if err != nil {
		t.Fatal(err)
	}
	other := e.task(t, project.ID, CreateTaskInput{Title: "Write docs", Description: "Line one\nline two"})
	comment, err := e.comments.Create(e.ctx, e.session, task.ID, "Looks urgent")
	if err != nil {
		t.Fatal(err)
	}
	label, err := e.labels.Create(e.ctx, e.session, project.ID, CreateLabelInput{Name: "bug", Color: "red"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{label.ID}
	if _, err := e.tasks.Update(e.ctx, e.session, task.ID, UpdateTaskInput{LabelIDs: &ids}); err != nil {
		t.Fatal(err)
	}
	notes := e.upload(t, "", "notes.md", "# Notes")
	if err := e.svc.Link(e.ctx, e.session, task.ID, notes.ID); err != nil {
		t.Fatal(err)
	}
	planID := e.planOf(t, project.ID)
	plan, err := e.projects.repo.GetPlan(e.ctx, e.db, e.session.WorkspaceID, project.ID, planID)
	if err != nil {
		t.Fatal(err)
	}
	id := project.Identifier

	t.Run("task", func(t *testing.T) {
		got, status, _ := e.text(t, attach(model.AIAttachTask, task.ID, "x"), aiTotalLimit)
		want := "<attachment kind=\"task\" key=\"" + id + "-1\" title=\"Fix login\">\n" +
			"Project: Backend · Plan: " + plan.Name + " · Status: To Do · Priority: high\n" +
			"Start: 2026-11-01 · Due: 2026-11-30 · Completed: no\n" +
			"Labels: bug\n" +
			"Description:\nUsers cannot log in.\n" +
			"Subtasks:\n- [ ] " + id + "-" + itoa(sub.Number) + " Reproduce\n" +
			"Comments (latest 20, oldest first):\n- " + comment.AuthorName + ", " + comment.CreatedAt.UTC().Format("2006-01-02") + ": Looks urgent\n" +
			"Linked files: notes.md\n" +
			"</attachment>"
		if got != want || status != model.AIAttachIncluded {
			t.Errorf("task = %q (%s)\nwant %q", got, status, want)
		}
	})
	t.Run("plan", func(t *testing.T) {
		got, _, _ := e.text(t, attach(model.AIAttachPlan, planID, "x"), aiTotalLimit)
		want := "<attachment kind=\"plan\" project=\"Backend\" name=\"" + plan.Name + "\">\n" +
			"- " + id + "-1 [To Do · high · due 2026-11-30] Fix login — Users cannot log in.\n" +
			"  - " + id + "-" + itoa(sub.Number) + " [To Do] Reproduce\n" +
			"- " + id + "-" + itoa(other.Number) + " [To Do] Write docs — Line one line two\n" +
			"</attachment>"
		if got != want {
			t.Errorf("plan = %q\nwant %q", got, want)
		}
	})
	t.Run("project", func(t *testing.T) {
		got, _, _ := e.text(t, attach(model.AIAttachProject, project.ID, "x"), aiTotalLimit)
		if !strings.HasPrefix(got, "<attachment kind=\"project\" name=\"Backend\">\n## Plan: "+plan.Name+"\n- "+id+"-1 [To Do") || !strings.HasSuffix(got, "</attachment>") {
			t.Errorf("project = %q", got)
		}
	})
	t.Run("missing", func(t *testing.T) {
		got, status, used := e.text(t, attach(model.AIAttachTask, model.NewID(), "BAC-9 Gone"), aiTotalLimit)
		if want := `<attachment kind="task" name="BAC-9 Gone" missing="true"/>`; got != want || status != model.AIAttachMissing || used != 0 {
			t.Errorf("missing = %q (%s)", got, status)
		}
	})
	t.Run("deleted task", func(t *testing.T) {
		if err := e.tasks.Delete(e.ctx, e.session, other.ID); err != nil {
			t.Fatal(err)
		}
		_, status, _ := e.text(t, attach(model.AIAttachTask, other.ID, "x"), aiTotalLimit)
		if status != model.AIAttachMissing {
			t.Errorf("status = %s", status)
		}
	})
}

func itoa(n int) string { return intString(n) }

func intString(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

func TestAIPromptFiles(t *testing.T) {
	e := newAIEnv(t)
	small := e.upload(t, "", "notes.md", "hello ş")
	bin := e.upload(t, "", "logo.bin", "\x00\x01\x02")
	folder := e.folder(t, "", "docs")
	big := e.upload(t, "", "big.txt", strings.Repeat("ş", 40_000)) // 80 000 bytes

	got, status, used := e.text(t, attach(model.AIAttachResource, small.ID, "notes.md"), aiTotalLimit)
	if want := "<attachment kind=\"file\" name=\"notes.md\">\nhello ş\n</attachment>"; got != want || status != model.AIAttachIncluded || used != len("hello ş") {
		t.Errorf("file = %q (%s, %d)", got, status, used)
	}
	if got, status, _ := e.text(t, attach(model.AIAttachResource, bin.ID, "logo.bin"), aiTotalLimit); status != model.AIAttachUnreadable || got != `<attachment kind="file" name="logo.bin" unreadable="true"/>` {
		t.Errorf("binary = %q (%s)", got, status)
	}
	if _, status, _ := e.text(t, attach(model.AIAttachResource, folder.ID, "docs"), aiTotalLimit); status != model.AIAttachUnreadable {
		t.Errorf("folder status = %s", status)
	}

	// Truncated at a rune boundary, within the per-attachment limit.
	got, status, used = e.text(t, attach(model.AIAttachResource, big.ID, "big.txt"), aiTotalLimit)
	body := strings.TrimSuffix(strings.TrimPrefix(got, "<attachment kind=\"file\" name=\"big.txt\">\n"), "\n</attachment>")
	if status != model.AIAttachTruncated || !strings.HasSuffix(body, "\n[truncated]") {
		t.Fatalf("big status = %s, tail %q", status, body[len(body)-20:])
	}
	kept := strings.TrimSuffix(body, "\n[truncated]")
	if len(kept) != aiAttachmentLimit || strings.ToValidUTF8(kept, "") != kept || used != len(body) {
		t.Errorf("kept %d bytes (valid %v), used %d", len(kept), strings.ToValidUTF8(kept, "") == kept, used)
	}

	// Small remaining budget cuts further; below the minimum it is omitted.
	_, status, _ = e.text(t, attach(model.AIAttachResource, big.ID, "big.txt"), 5000)
	if status != model.AIAttachTruncated {
		t.Errorf("status = %s, want truncated", status)
	}
	got, status, used = e.text(t, attach(model.AIAttachResource, big.ID, "big.txt"), aiMinBudget-1)
	if want := `<attachment kind="file" name="big.txt" omitted="too much context"/>`; got != want || status != model.AIAttachOmitted || used != 0 {
		t.Errorf("omitted = %q (%s)", got, status)
	}

	// Trashed target counts as missing.
	if err := e.svc.MoveToTrash(e.ctx, e.session, small.ID); err != nil {
		t.Fatal(err)
	}
	if _, status, _ := e.text(t, attach(model.AIAttachResource, small.ID, "notes.md"), aiTotalLimit); status != model.AIAttachMissing {
		t.Errorf("trashed status = %s", status)
	}
	// A file whose bytes vanished is unreadable.
	if err := os.Remove(e.disk(big.Name)); err != nil {
		t.Fatal(err)
	}
	if _, status, _ := e.text(t, attach(model.AIAttachResource, big.ID, "big.txt"), aiTotalLimit); status != model.AIAttachUnreadable {
		t.Errorf("vanished status = %s", status)
	}
}

func TestAIPromptRepoFile(t *testing.T) {
	e := newAIEnv(t)
	repo := e.addHello(t, "")
	got, status, _ := e.text(t, model.AIAttachment{ID: "a", Kind: model.AIAttachRepoFile, TargetID: repo.ID, Path: "src/main.go", Label: "x"}, aiTotalLimit)
	if want := "<attachment kind=\"file\" repository=\"octo/hello\" path=\"src/main.go\">\npackage main\n</attachment>"; got != want || status != model.AIAttachIncluded {
		t.Errorf("repo file = %q (%s)", got, status)
	}
	if _, status, _ := e.text(t, model.AIAttachment{ID: "b", Kind: model.AIAttachRepoFile, TargetID: repo.ID, Path: "logo.bin", Label: "x"}, aiTotalLimit); status != model.AIAttachUnreadable {
		t.Errorf("binary status = %s", status)
	}
	if _, status, _ := e.text(t, model.AIAttachment{ID: "c", Kind: model.AIAttachRepoFile, TargetID: repo.ID, Path: "nope.go", Label: "x"}, aiTotalLimit); status != model.AIAttachMissing {
		t.Errorf("missing status = %s", status)
	}
}

func TestAIPromptBudgetKeepsNewest(t *testing.T) {
	e := newAIEnv(t)
	// Five files of 64 KiB: only 256 KiB fit, so the oldest message loses.
	var files []model.Resource
	for i := 0; i < 5; i++ {
		files = append(files, e.upload(t, "", "f"+intString(i)+".txt", strings.Repeat("a", aiAttachmentLimit)))
	}
	now := time.Now()
	var msgs []model.AIMessage
	atts := map[string][]model.AIAttachment{}
	for i, f := range files {
		m := model.AIMessage{ID: "m" + intString(i), Role: model.AIRoleUser, Content: "q" + intString(i), Status: model.AIStatusComplete, CreatedAt: now}
		msgs = append(msgs, m)
		a := attach(model.AIAttachResource, f.ID, f.Name)
		a.ID = "a" + intString(i)
		atts[m.ID] = []model.AIAttachment{a}
	}
	_, statuses, err := e.prompt.build(e.ctx, e.session, msgs, atts)
	if err != nil {
		t.Fatal(err)
	}
	// 4 * (64 KiB + marker-free) = 256 KiB: the 4 newest fit exactly, the oldest is omitted.
	want := map[string]model.AIAttachmentStatus{"a4": model.AIAttachIncluded, "a3": model.AIAttachIncluded, "a2": model.AIAttachIncluded, "a1": model.AIAttachIncluded, "a0": model.AIAttachOmitted}
	for id, w := range want {
		if statuses[id] != w {
			t.Errorf("status[%s] = %s, want %s", id, statuses[id], w)
		}
	}
}

func TestAIPromptMessagesAndInstructions(t *testing.T) {
	e := newAIEnv(t)
	msgs := []model.AIMessage{
		{ID: "1", Role: model.AIRoleUser, Content: "hi", Status: model.AIStatusComplete},
		{ID: "2", Role: model.AIRoleAssistant, Content: "partial", Status: model.AIStatusError},
		{ID: "3", Role: model.AIRoleUser, Content: "again", Status: model.AIStatusComplete},
		{ID: "4", Role: model.AIRoleAssistant, Content: "  ", Status: model.AIStatusStopped},
		{ID: "5", Role: model.AIRoleAssistant, Content: "hello", Status: model.AIStatusStopped},
	}
	got, _, err := e.prompt.build(context.Background(), e.session, msgs, nil)
	if err != nil || len(got) != 3 || got[0].Content != "hi" || got[1].Content != "again" || got[2].Content != "hello" {
		t.Errorf("build() = %+v (err %v)", got, err)
	}
	if aiSystem("") != aiSystemPrompt {
		t.Error("system prompt changed without instructions")
	}
	want := aiSystemPrompt + "\n\nThe user's personal instructions:\n<instructions>\nBe brief.\n</instructions>"
	if got := aiSystem(" Be brief. "); got != want {
		t.Errorf("aiSystem() = %q", got)
	}
}

func TestAIPromptUserMessageHasAttachmentsBeforeText(t *testing.T) {
	e := newAIEnv(t)
	f := e.upload(t, "", "a.txt", "A")
	msgs := []model.AIMessage{{ID: "1", Role: model.AIRoleUser, Content: "question", Status: model.AIStatusComplete}}
	a := attach(model.AIAttachResource, f.ID, "a.txt")
	got, _, err := e.prompt.build(e.ctx, e.session, msgs, map[string][]model.AIAttachment{"1": {a}})
	want := "<attachment kind=\"file\" name=\"a.txt\">\nA\n</attachment>\n\nquestion"
	if err != nil || len(got) != 1 || got[0].Content != want {
		t.Errorf("build() = %+v (err %v), want %q", got, err, want)
	}
}
