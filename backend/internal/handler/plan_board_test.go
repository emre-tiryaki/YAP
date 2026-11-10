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
	b.createTask(t, env, `{"title":"open"}`)
	b.createTask(t, env, `{"title":"finished","plan_id":"`+b.plan.ID+`","status_id":"`+done.ID+`"}`)

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

	rec := env.do(t, http.MethodPost, b.path("/statuses"), `{"title":"Review","color":"#a855f7","category":"in_progress"}`)
	wantStatus(t, rec, http.StatusCreated)
	created := decode[statusResponse](t, rec)

	statuses := decode[[]statusResponse](t, env.do(t, http.MethodGet, b.path("/statuses"), ""))
	if last := statuses[len(statuses)-1]; last != created || last.Title != "Review" {
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
