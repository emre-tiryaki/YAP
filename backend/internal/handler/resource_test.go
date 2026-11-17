package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func (e testEnv) raw(t *testing.T, method, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rec := httptest.NewRecorder()
	e.mux.ServeHTTP(rec, req)
	return rec
}

func TestResourceUploadServeAndSave(t *testing.T) {
	e := newTestEnv(t)

	rec := e.raw(t, "POST", "/api/resources/upload?name=page.html", "text/html", "<script>alert(1)</script>")
	wantStatus(t, rec, http.StatusCreated)
	file := decode[resourceResponse](t, rec)
	if file.ParentID != nil || file.Kind != "file" || !file.IsText || file.MimeType != "text/html" || file.Revision != 1 {
		t.Fatalf("uploaded = %+v", file)
	}

	rec = e.do(t, "GET", "/api/resources/"+file.ID+"/content", "")
	wantStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Security-Policy"); !strings.HasPrefix(got, "sandbox") {
		t.Errorf("CSP = %q, want a sandbox", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}
	if rec.Body.String() != "<script>alert(1)</script>" {
		t.Errorf("body = %q", rec.Body.String())
	}

	rec = e.do(t, "GET", "/api/resources/"+file.ID+"/content?download=1", "")
	if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, "attachment") {
		t.Errorf("Content-Disposition = %q, want attachment", got)
	}

	rec = e.raw(t, "PUT", "/api/resources/"+file.ID+"/content?revision=1", "text/plain", "<p>hi</p>")
	wantStatus(t, rec, http.StatusOK)
	if got := decode[resourceResponse](t, rec).Revision; got != 2 {
		t.Errorf("revision = %d, want 2", got)
	}
	rec = e.raw(t, "PUT", "/api/resources/"+file.ID+"/content?revision=1", "text/plain", "old")
	wantErrorCode(t, rec, http.StatusConflict, "stale")
	rec = e.raw(t, "PUT", "/api/resources/"+file.ID+"/content", "text/plain", "x")
	wantErrorCode(t, rec, http.StatusBadRequest, "invalid_input")
}

func TestResourceTreeRoutes(t *testing.T) {
	e := newTestEnv(t)

	rec := e.do(t, "POST", "/api/resources/folders", `{"parent_id":null,"name":"docs"}`)
	wantStatus(t, rec, http.StatusCreated)
	docs := decode[resourceResponse](t, rec)

	rec = e.do(t, "POST", "/api/resources/files", `{"parent_id":"`+docs.ID+`","name":"a.md"}`)
	wantStatus(t, rec, http.StatusCreated)
	file := decode[resourceResponse](t, rec)

	rec = e.do(t, "POST", "/api/resources/folders", `{"name":"DOCS"}`)
	wantErrorCode(t, rec, http.StatusConflict, "conflict")

	rec = e.do(t, "PATCH", "/api/resources/"+file.ID, `{"name":"b.md"}`)
	wantStatus(t, rec, http.StatusOK)

	rec = e.do(t, "POST", "/api/resources/"+file.ID+"/copy", `{"parent_id":"`+docs.ID+`","on_conflict":"keep_both"}`)
	wantStatus(t, rec, http.StatusCreated)
	if got := decode[resourceResponse](t, rec).Name; got != "b (1).md" {
		t.Errorf("copy name = %q", got)
	}

	rec = e.do(t, "POST", "/api/resources/"+file.ID+"/move", `{"parent_id":null}`)
	wantStatus(t, rec, http.StatusOK)

	rec = e.do(t, "GET", "/api/resources", "")
	wantStatus(t, rec, http.StatusOK)
	if got := len(decode[[]resourceResponse](t, rec)); got != 3 {
		t.Errorf("listed = %d, want 3", got)
	}

	wantStatus(t, e.do(t, "POST", "/api/resources/"+docs.ID+"/trash", ""), http.StatusNoContent)
	rec = e.do(t, "GET", "/api/resources/trash", "")
	if got := decode[[]resourceResponse](t, rec); len(got) != 1 || got[0].TrashedAt == nil {
		t.Errorf("trash = %+v", got)
	}
	wantStatus(t, e.do(t, "POST", "/api/resources/"+docs.ID+"/restore", ""), http.StatusOK)
	wantStatus(t, e.do(t, "POST", "/api/resources/"+docs.ID+"/trash", ""), http.StatusNoContent)
	wantStatus(t, e.do(t, "DELETE", "/api/resources/trash", ""), http.StatusNoContent)

	rec = e.do(t, "GET", "/api/resources/events?limit=3", "")
	wantStatus(t, rec, http.StatusOK)
	events := decode[[]resourceEventResponse](t, rec)
	if len(events) != 3 || events[0].Action != "purged" || events[0].ActorName != "tester" {
		t.Errorf("events = %+v", events)
	}
	rec = e.do(t, "GET", "/api/resources/"+docs.ID+"/events", "")
	wantStatus(t, rec, http.StatusOK)
}

func TestResourceTaskLinkRoutes(t *testing.T) {
	e := newTestEnv(t)
	project := decode[projectResponse](t, e.do(t, "POST", "/api/projects", `{"name":"P"}`))
	rec := e.do(t, "POST", "/api/projects/"+project.ID+"/tasks", `{"title":"T"}`)
	wantStatus(t, rec, http.StatusCreated)
	task := decode[struct {
		ID string `json:"id"`
	}](t, rec)
	file := decode[resourceResponse](t, e.raw(t, "POST", "/api/resources/upload?name=a.txt", "", "x"))

	wantStatus(t, e.do(t, "PUT", "/api/tasks/"+task.ID+"/resources/"+file.ID, ""), http.StatusNoContent)
	rec = e.do(t, "GET", "/api/tasks/"+task.ID+"/resources", "")
	if got := decode[[]resourceResponse](t, rec); len(got) != 1 || got[0].ID != file.ID {
		t.Errorf("task resources = %+v", got)
	}
	rec = e.do(t, "GET", "/api/resources/"+file.ID+"/tasks", "")
	if got := decode[[]linkedTaskResponse](t, rec); len(got) != 1 || got[0].ProjectName != "P" {
		t.Errorf("resource tasks = %+v", got)
	}
	rec = e.do(t, "GET", "/api/projects/"+project.ID+"/resource-events", "")
	if got := decode[[]resourceEventResponse](t, rec); len(got) != 2 || got[0].Action != "linked" || got[0].TaskID == nil {
		t.Errorf("project events = %+v", got)
	}
	wantStatus(t, e.do(t, "DELETE", "/api/tasks/"+task.ID+"/resources/"+file.ID, ""), http.StatusNoContent)
	wantErrorCode(t, e.do(t, "PUT", "/api/tasks/"+task.ID+"/resources/missing", ""), http.StatusNotFound, "not_found")
}

func TestSettingsRoutes(t *testing.T) {
	e := newTestEnv(t)
	rec := e.do(t, "GET", "/api/settings", "")
	wantStatus(t, rec, http.StatusOK)
	if got := decode[settingsResponse](t, rec); !got.AutosaveEnabled || got.AutosaveDelayMs != 1000 {
		t.Errorf("defaults = %+v", got)
	}
	rec = e.do(t, "PATCH", "/api/settings", `{"autosave_enabled":false}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[settingsResponse](t, rec); got.AutosaveEnabled {
		t.Errorf("after patch = %+v", got)
	}
	wantErrorCode(t, e.do(t, "PATCH", "/api/settings", `{"autosave_delay_ms":5}`), http.StatusBadRequest, "invalid_input")

	rec = e.do(t, "PATCH", "/api/settings",
		`{"hide_done":true,"status_names":{"done":"Shipped"},"notifications":{"reminder":"1w","comments":true}}`)
	wantStatus(t, rec, http.StatusOK)
	got := decode[settingsResponse](t, rec)
	if !got.HideDone || got.DoneToBottom || got.StatusNames.Done != "Shipped" || got.StatusNames.ToDo != "To Do" {
		t.Errorf("after task view patch = %+v", got)
	}
	if n := got.Notifications; n.Reminder != "1w" || !n.Comments || !n.DueSoon || !n.Overdue || n.Sound {
		t.Errorf("notifications = %+v", n)
	}
	wantErrorCode(t, e.do(t, "PATCH", "/api/settings", `{"notifications":{"reminder":"soon"}}`), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.do(t, "PATCH", "/api/settings", `{"status_names":{"blocked":"x"}}`), http.StatusBadRequest, "invalid_input")
}

func TestRecentCommentsRoute(t *testing.T) {
	e := newTestEnv(t)
	rec := e.do(t, "GET", "/api/comments/recent?since=2026-01-01T00:00:00Z", "")
	wantStatus(t, rec, http.StatusOK)
	if got := decode[[]recentCommentResponse](t, rec); len(got) != 0 {
		t.Errorf("recent = %+v, want none (own comments only)", got)
	}
	wantErrorCode(t, e.do(t, "GET", "/api/comments/recent", ""), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.do(t, "GET", "/api/comments/recent?since=yesterday", ""), http.StatusBadRequest, "invalid_input")
}
