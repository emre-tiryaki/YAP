package handler

import (
	"net/http"
	"testing"
)

type planBoard struct {
	project  projectResponse
	plan     planResponse
	statuses []statusResponse
}

func newPlanBoard(t *testing.T, env testEnv) planBoard {
	t.Helper()
	project := createProject(t, env, "Alpha")
	plans := decode[[]planResponse](t, env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans", ""))
	plan := plans[0]
	rec := env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans/"+plan.ID+"/statuses", "")
	wantStatus(t, rec, http.StatusOK)
	return planBoard{project: project, plan: plan, statuses: decode[[]statusResponse](t, rec)}
}

func (b planBoard) path(suffix string) string {
	return "/api/projects/" + b.project.ID + "/plans/" + b.plan.ID + suffix
}

func (b planBoard) createTask(t *testing.T, env testEnv, body string) taskResponse {
	t.Helper()
	rec := env.do(t, http.MethodPost, "/api/projects/"+b.project.ID+"/tasks", body)
	wantStatus(t, rec, http.StatusCreated)
	return decode[taskResponse](t, rec)
}

func TestPlanGetReportsCounts(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	done := b.statuses[2]
	// A task in the done column is not finished until it is marked done.
	b.createTask(t, env, `{"title":"parked","plan_id":"`+b.plan.ID+`","status_id":"`+done.ID+`"}`)
	finished := b.createTask(t, env, `{"title":"finished"}`)
	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/"+finished.ID+"/done", `{"done":true}`), http.StatusOK)

	rec := env.do(t, http.MethodGet, b.path(""), "")
	wantStatus(t, rec, http.StatusOK)
	got := decode[planResponse](t, rec)
	if got.ID != b.plan.ID || got.Name != "Main" || got.TaskCount != 2 || got.DoneCount != 1 {
		t.Errorf("plan = %+v, want Main with 2 tasks, 1 done", got)
	}
}

func TestPlanGetNotFound(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	for _, path := range []string{
		"/api/projects/" + b.project.ID + "/plans/missing",
		"/api/projects/missing/plans/" + b.plan.ID,
		"/api/projects/" + b.project.ID + "/plans/missing/statuses",
		"/api/projects/" + b.project.ID + "/plans/missing/tasks",
	} {
		t.Run(path, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodGet, path, ""), http.StatusNotFound, "not_found")
		})
	}
}

func TestPlanStatusesDefaultOrder(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)

	want := []string{"todo", "in_progress", "done"}
	if len(b.statuses) != len(want) {
		t.Fatalf("statuses = %+v, want 3", b.statuses)
	}
	for i, s := range b.statuses {
		if s.Category != want[i] || s.PlanID != b.plan.ID {
			t.Errorf("status %d = %+v, want category %q", i, s, want[i])
		}
	}
}

func TestPlanCreateStatus(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)

	rec := env.do(t, http.MethodPost, b.path("/statuses"), `{"title":"Review","description":"Awaiting review","color":"#a855f7","category":"in_progress"}`)
	wantStatus(t, rec, http.StatusCreated)
	created := decode[statusResponse](t, rec)

	statuses := decode[[]statusResponse](t, env.do(t, http.MethodGet, b.path("/statuses"), ""))
	if last := statuses[len(statuses)-1]; last != created || last.Title != "Review" || last.Description != "Awaiting review" {
		t.Errorf("last status = %+v, want created %+v", last, created)
	}
}

func TestPlanCreateStatusInvalid(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)

	for name, body := range map[string]string{
		"unknown category": `{"title":"x","color":"#000000","category":"blocked"}`,
		"missing title":    `{"title":"","color":"#000000","category":"todo"}`,
		"bad color":        `{"title":"x","color":"blue","category":"todo"}`,
		"unknown field":    `{"title":"x","color":"#000000","category":"todo","extra":1}`,
	} {
		t.Run(name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodPost, b.path("/statuses"), body), http.StatusBadRequest, "invalid_input")
		})
	}
}

func TestPlanTasksMoveAndDelete(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	first := b.createTask(t, env, `{"title":"first","description":"desc","due_date":"2026-12-01T00:00:00Z"}`)
	second := b.createTask(t, env, `{"title":"second"}`)
	inProgress := b.statuses[1]

	rec := env.do(t, http.MethodGet, b.path("/tasks"), "")
	wantStatus(t, rec, http.StatusOK)
	tasks := decode[[]planTaskResponse](t, rec)
	if len(tasks) != 2 || tasks[0].ID != first.ID || tasks[1].ID != second.ID {
		t.Fatalf("tasks = %+v, want first then second", tasks)
	}
	if tasks[0].Description != "desc" || tasks[0].DueDate == nil || *tasks[0].DueDate != "2026-12-01T00:00:00Z" {
		t.Errorf("first = %+v", tasks[0])
	}

	rec = env.do(t, http.MethodPost, "/api/tasks/"+second.ID+"/move", `{"status_id":"`+inProgress.ID+`","index":0}`)
	wantStatus(t, rec, http.StatusNoContent)

	tasks = decode[[]planTaskResponse](t, env.do(t, http.MethodGet, b.path("/tasks"), ""))
	if len(tasks) != 2 || tasks[1].ID != second.ID || tasks[1].StatusID != inProgress.ID || tasks[1].StatusCategory != "in_progress" {
		t.Errorf("after move tasks = %+v", tasks)
	}

	wantStatus(t, env.do(t, http.MethodDelete, "/api/tasks/"+first.ID, ""), http.StatusNoContent)
	wantErrorCode(t, env.do(t, http.MethodDelete, "/api/tasks/"+first.ID, ""), http.StatusNotFound, "not_found")

	tasks = decode[[]planTaskResponse](t, env.do(t, http.MethodGet, b.path("/tasks"), ""))
	if len(tasks) != 1 || tasks[0].ID != second.ID {
		t.Errorf("after delete tasks = %+v", tasks)
	}
}

func TestTaskMoveErrors(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	task := b.createTask(t, env, `{"title":"x"}`)

	tests := []struct {
		name, path, body string
		status           int
		code             string
	}{
		{"missing task", "/api/tasks/missing/move", `{"status_id":"` + b.statuses[0].ID + `"}`, http.StatusNotFound, "not_found"},
		{"missing status", "/api/tasks/" + task.ID + "/move", `{"index":0}`, http.StatusBadRequest, "invalid_input"},
		{"unknown status", "/api/tasks/" + task.ID + "/move", `{"status_id":"missing"}`, http.StatusBadRequest, "invalid_input"},
		{"malformed body", "/api/tasks/" + task.ID + "/move", `{`, http.StatusBadRequest, "invalid_input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodPost, tt.path, tt.body), tt.status, tt.code)
		})
	}
}

func TestPlanMoveStatus(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	done := b.statuses[2]

	rec := env.do(t, http.MethodPost, b.path("/statuses/"+done.ID+"/move"), `{"index":0}`)
	wantStatus(t, rec, http.StatusNoContent)

	statuses := decode[[]statusResponse](t, env.do(t, http.MethodGet, b.path("/statuses"), ""))
	got := []string{}
	for _, s := range statuses {
		got = append(got, s.Category)
	}
	want := []string{"done", "todo", "in_progress"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestPlanMoveStatusErrors(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	todo := b.statuses[0]

	tests := []struct {
		name, path, body string
		status           int
		code             string
	}{
		{"missing status", b.path("/statuses/missing/move"), `{"index":0}`, http.StatusNotFound, "not_found"},
		{"missing plan", "/api/projects/" + b.project.ID + "/plans/missing/statuses/" + todo.ID + "/move", `{"index":0}`, http.StatusNotFound, "not_found"},
		{"malformed body", b.path("/statuses/" + todo.ID + "/move"), `{`, http.StatusBadRequest, "invalid_input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodPost, tt.path, tt.body), tt.status, tt.code)
		})
	}
}

func TestPlanUpdateStatus(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	todo := b.statuses[0]

	rec := env.do(t, http.MethodPatch, b.path("/statuses/"+todo.ID), `{"title":"Backlog","description":"Ideas","color":"#EF4444"}`)
	wantStatus(t, rec, http.StatusOK)
	updated := decode[statusResponse](t, rec)
	if updated.Title != "Backlog" || updated.Description != "Ideas" || updated.Color != "#ef4444" || updated.Category != todo.Category {
		t.Errorf("updated = %+v", updated)
	}

	statuses := decode[[]statusResponse](t, env.do(t, http.MethodGet, b.path("/statuses"), ""))
	if statuses[0] != updated {
		t.Errorf("stored = %+v, want %+v", statuses[0], updated)
	}
}

func TestPlanUpdateStatusErrors(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	todo := b.statuses[0]

	tests := []struct {
		name, path, body string
		status           int
		code             string
	}{
		{"missing status", b.path("/statuses/missing"), `{"title":"x"}`, http.StatusNotFound, "not_found"},
		{"missing plan", "/api/projects/" + b.project.ID + "/plans/missing/statuses/" + todo.ID, `{"title":"x"}`, http.StatusNotFound, "not_found"},
		{"empty body", b.path("/statuses/" + todo.ID), `{}`, http.StatusBadRequest, "invalid_input"},
		{"bad color", b.path("/statuses/" + todo.ID), `{"color":"blue"}`, http.StatusBadRequest, "invalid_input"},
		{"unknown field", b.path("/statuses/" + todo.ID), `{"category":"done"}`, http.StatusBadRequest, "invalid_input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodPatch, tt.path, tt.body), tt.status, tt.code)
		})
	}
}

func TestPlanDeleteStatus(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	todo := b.statuses[0]
	b.createTask(t, env, `{"title":"gone","plan_id":"`+b.plan.ID+`","status_id":"`+todo.ID+`"}`)

	wantStatus(t, env.do(t, http.MethodDelete, b.path("/statuses/"+todo.ID), ""), http.StatusNoContent)

	statuses := decode[[]statusResponse](t, env.do(t, http.MethodGet, b.path("/statuses"), ""))
	if len(statuses) != 2 || statuses[0].ID == todo.ID {
		t.Errorf("statuses = %+v, want todo removed", statuses)
	}
	got := decode[planResponse](t, env.do(t, http.MethodGet, b.path(""), ""))
	if got.TaskCount != 0 {
		t.Errorf("task count = %d, want 0", got.TaskCount)
	}
}

func TestPlanDeleteStatusErrors(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	done := b.statuses[2]

	tests := []struct {
		name, path string
		status     int
		code       string
	}{
		{"last done status", b.path("/statuses/" + done.ID), http.StatusConflict, "conflict"},
		{"missing status", b.path("/statuses/missing"), http.StatusNotFound, "not_found"},
		{"missing plan", "/api/projects/" + b.project.ID + "/plans/missing/statuses/" + done.ID, http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodDelete, tt.path, ""), tt.status, tt.code)
		})
	}
}
