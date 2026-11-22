package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func taskActions(t *testing.T, env testEnv, taskID string) []string {
	t.Helper()
	rec := env.do(t, http.MethodGet, "/api/tasks/"+taskID+"/activity", "")
	wantStatus(t, rec, http.StatusOK)
	var actions []string
	for _, e := range decode[[]taskEventResponse](t, rec) {
		actions = append(actions, e.Action)
	}
	return actions
}

func sameActions(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestTaskActivityOverHTTP(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Backend")

	rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"T"}`)
	wantStatus(t, rec, http.StatusCreated)
	task := decode[taskResponse](t, rec)

	wantStatus(t, env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"title":"T2","priority":"high"}`), http.StatusOK)
	wantStatus(t, env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"start_date":"2026-12-01T00:00:00Z","due_date":"2026-12-05T00:00:00Z"}`), http.StatusOK)
	// An unchanged value writes no event.
	wantStatus(t, env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"priority":"high"}`), http.StatusOK)
	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/"+task.ID+"/done", `{"done":true}`), http.StatusOK)
	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/"+task.ID+"/done", `{"done":false}`), http.StatusOK)

	got := taskActions(t, env, task.ID)
	want := []string{"created", "renamed", "priority_changed", "due_changed", "start_changed", "done", "reopened"}
	// Events of one request share a second; compare as sets in that case.
	if len(got) != len(want) || got[0] != "created" || got[len(got)-2] != "done" || got[len(got)-1] != "reopened" {
		t.Errorf("actions = %v, want %v", got, want)
	}

	rec = env.do(t, http.MethodGet, "/api/tasks/"+task.ID+"/activity", "")
	for _, e := range decode[[]taskEventResponse](t, rec) {
		if e.Action == "renamed" && (e.Details["from"] != "T" || e.Details["to"] != "T2") {
			t.Errorf("renamed details = %v", e.Details)
		}
	}

	wantStatus(t, env.do(t, http.MethodGet, "/api/tasks/missing/activity", ""), http.StatusNotFound)
}

func TestTaskStatusChangeEvent(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Backend")
	rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"T"}`)
	task := decode[taskResponse](t, rec)

	rec = env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans/"+task.PlanID+"/statuses", "")
	wantStatus(t, rec, http.StatusOK)
	statuses := decode[[]statusResponse](t, rec)
	var other string
	for _, s := range statuses {
		if s.ID != task.StatusID {
			other = s.ID
			break
		}
	}
	if other == "" {
		t.Fatal("plan has only one status")
	}
	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/"+task.ID+"/move", `{"status_id":"`+other+`","index":0}`), http.StatusNoContent)
	// Reordering inside the same status writes no event.
	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/"+task.ID+"/move", `{"status_id":"`+other+`","index":0}`), http.StatusNoContent)

	rec = env.do(t, http.MethodGet, "/api/tasks/"+task.ID+"/activity", "")
	var changes int
	for _, e := range decode[[]taskEventResponse](t, rec) {
		if e.Action == "status_changed" {
			changes++
			if e.Details["from"] == "" || e.Details["to"] == "" || e.Details["from"] == e.Details["to"] {
				t.Errorf("status details = %v, want two different titles", e.Details)
			}
		}
	}
	if changes != 1 {
		t.Errorf("status_changed events = %d, want 1", changes)
	}
}

func TestTaskStartDate(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Backend")

	tests := []struct {
		name string
		body string
		want int
	}{
		{"start before due", `{"title":"a","start_date":"2026-12-01T00:00:00Z","due_date":"2026-12-02T00:00:00Z"}`, http.StatusCreated},
		{"start equals due", `{"title":"b","start_date":"2026-12-01T00:00:00Z","due_date":"2026-12-01T00:00:00Z"}`, http.StatusCreated},
		{"start without due", `{"title":"c","start_date":"2026-12-01T00:00:00Z"}`, http.StatusCreated},
		{"start after due", `{"title":"d","start_date":"2026-12-03T00:00:00Z","due_date":"2026-12-02T00:00:00Z"}`, http.StatusBadRequest},
		{"bad start", `{"title":"e","start_date":"soon"}`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", tt.body)
			wantStatus(t, rec, tt.want)
		})
	}

	rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"f","due_date":"2026-12-05T00:00:00Z"}`)
	task := decode[taskResponse](t, rec)
	wantStatus(t, env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"start_date":"2026-12-06T00:00:00Z"}`), http.StatusBadRequest)
	rec = env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"start_date":"2026-12-04T00:00:00Z"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[planTaskResponse](t, rec); got.StartDate == nil || *got.StartDate != "2026-12-04T00:00:00Z" {
		t.Errorf("start_date = %v", got.StartDate)
	}
	// Moving the due date before the start is rejected too; null clears the start.
	wantStatus(t, env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"due_date":"2026-12-03T00:00:00Z"}`), http.StatusBadRequest)
	rec = env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"start_date":null}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[planTaskResponse](t, rec); got.StartDate != nil {
		t.Errorf("start_date after clear = %v, want nil", *got.StartDate)
	}
}

func TestTaskRelationsOverHTTP(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Backend")
	other := createProject(t, env, "Frontend")
	create := func(projectID, title string) taskResponse {
		rec := env.do(t, http.MethodPost, "/api/projects/"+projectID+"/tasks", `{"title":"`+title+`"}`)
		wantStatus(t, rec, http.StatusCreated)
		return decode[taskResponse](t, rec)
	}
	a, b, c := create(project.ID, "A"), create(project.ID, "B"), create(other.ID, "C")

	add := func(from, to, kind string) *httptest.ResponseRecorder {
		return env.do(t, http.MethodPost, "/api/tasks/"+from+"/relations", `{"related_task_id":"`+to+`","kind":"`+kind+`"}`)
	}
	wantStatus(t, add(a.ID, b.ID, "blocks"), http.StatusNoContent)
	// Across projects of the workspace is fine.
	wantStatus(t, add(a.ID, c.ID, "relates"), http.StatusNoContent)

	tests := []struct {
		name string
		from string
		to   string
		kind string
		want int
	}{
		{"same pair again", a.ID, b.ID, "blocks", http.StatusConflict},
		{"reverse pair", b.ID, a.ID, "blocks", http.StatusConflict},
		{"reverse pair other kind", b.ID, a.ID, "duplicates", http.StatusConflict},
		{"self", a.ID, a.ID, "blocks", http.StatusBadRequest},
		{"unknown kind", b.ID, c.ID, "loves", http.StatusBadRequest},
		{"missing related", b.ID, "missing", "blocks", http.StatusNotFound},
		{"missing task", "missing", b.ID, "blocks", http.StatusNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantStatus(t, add(tt.from, tt.to, tt.kind), tt.want)
		})
	}

	views := func(taskID string) map[string]string {
		rec := env.do(t, http.MethodGet, "/api/tasks/"+taskID+"/relations", "")
		wantStatus(t, rec, http.StatusOK)
		out := map[string]string{}
		for _, r := range decode[[]taskRelationResponse](t, rec) {
			out[r.TaskID] = r.View
		}
		return out
	}
	if got := views(a.ID); got[b.ID] != "blocking" || got[c.ID] != "relates" {
		t.Errorf("relations of A = %v", got)
	}
	if got := views(b.ID); got[a.ID] != "blocked_by" {
		t.Errorf("relations of B = %v", got)
	}
	if got := views(c.ID); got[a.ID] != "relates" {
		t.Errorf("relations of C = %v, want the symmetric relates", got)
	}

	blocked := func(task taskResponse) bool {
		rec := env.do(t, http.MethodGet, "/api/projects/"+task.ProjectID+"/plans/"+task.PlanID+"/tasks", "")
		for _, x := range decode[[]planTaskResponse](t, rec) {
			if x.ID == task.ID {
				return x.Blocked
			}
		}
		t.Fatalf("task %s not listed", task.ID)
		return false
	}
	if !blocked(b) || blocked(a) {
		t.Errorf("blocked: B = %v, A = %v, want B blocked only", blocked(b), blocked(a))
	}
	// Finishing the blocker unblocks the task.
	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/"+a.ID+"/done", `{"done":true}`), http.StatusOK)
	if blocked(b) {
		t.Errorf("B still blocked after the blocker is done")
	}
	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/"+a.ID+"/done", `{"done":false}`), http.StatusOK)

	if got := taskActions(t, env, b.ID); got[len(got)-1] != "relation_added" {
		t.Errorf("actions of B = %v, want relation_added last", got)
	}

	// Removing works from either side and is logged on both tasks.
	wantStatus(t, env.do(t, http.MethodDelete, "/api/tasks/"+b.ID+"/relations/"+a.ID, ""), http.StatusNoContent)
	wantStatus(t, env.do(t, http.MethodDelete, "/api/tasks/"+b.ID+"/relations/"+a.ID, ""), http.StatusNotFound)
	if got := views(a.ID); len(got) != 1 || got[c.ID] != "relates" {
		t.Errorf("relations of A after removal = %v", got)
	}
	if got := taskActions(t, env, a.ID); got[len(got)-1] != "relation_removed" {
		t.Errorf("actions of A = %v, want relation_removed last", got)
	}
	// The pair can be related again, the other way round.
	wantStatus(t, add(b.ID, a.ID, "blocks"), http.StatusNoContent)

	// Deleting a task closes its relations.
	wantStatus(t, env.do(t, http.MethodDelete, "/api/tasks/"+c.ID, ""), http.StatusNoContent)
	if got := views(a.ID); len(got) != 1 || got[b.ID] != "blocked_by" {
		t.Errorf("relations of A after deleting C = %v", got)
	}
}
