package handler

import (
	"net/http"
	"testing"
)

func TestProjectGet(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha","description":"first"}`)
	wantStatus(t, rec, http.StatusCreated)
	created := decode[projectResponse](t, rec)

	rec = env.do(t, http.MethodGet, "/api/projects/"+created.ID, "")
	wantStatus(t, rec, http.StatusOK)
	got := decode[projectResponse](t, rec)
	if got != created {
		t.Errorf("project = %+v, want %+v", got, created)
	}
}

func TestProjectGetReportsTaskCount(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Alpha")

	for _, title := range []string{"one", "two"} {
		wantStatus(t, env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"`+title+`"}`), http.StatusCreated)
	}

	got := decode[projectResponse](t, env.do(t, http.MethodGet, "/api/projects/"+project.ID, ""))
	if got.TaskCount != 2 {
		t.Errorf("TaskCount = %d, want 2", got.TaskCount)
	}
}

func TestProjectGetNotFound(t *testing.T) {
	env := newTestEnv(t)
	wantErrorCode(t, env.do(t, http.MethodGet, "/api/projects/missing", ""), http.StatusNotFound, "not_found")
}

func TestProjectPlans(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Alpha")
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"one"}`), http.StatusCreated)

	rec := env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans", "")
	wantStatus(t, rec, http.StatusOK)
	got := decode[[]planResponse](t, rec)

	if len(got) != 1 {
		t.Fatalf("plans = %+v, want 1 default plan", got)
	}
	plan := got[0]
	if plan.ID == "" || plan.ProjectID != project.ID || plan.Name != "Main" {
		t.Errorf("plan = %+v", plan)
	}
	if plan.TeamID != nil || plan.TeamName != nil {
		t.Errorf("team = %v %v, want null for whole-project plan", plan.TeamID, plan.TeamName)
	}
	if plan.TaskCount != 1 {
		t.Errorf("TaskCount = %d, want 1", plan.TaskCount)
	}
}

func TestProjectPlansNotFound(t *testing.T) {
	env := newTestEnv(t)
	wantErrorCode(t, env.do(t, http.MethodGet, "/api/projects/missing/plans", ""), http.StatusNotFound, "not_found")
}

func TestProjectDetailMethodNotAllowed(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Alpha")

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			wantStatus(t, env.do(t, method, "/api/projects/"+project.ID, ""), http.StatusMethodNotAllowed)
		})
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run("plans "+method, func(t *testing.T) {
			wantStatus(t, env.do(t, method, "/api/projects/"+project.ID+"/plans", ""), http.StatusMethodNotAllowed)
		})
	}
}

func TestProjectCreatePlan(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Alpha")

	rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/plans", `{"name":"  Sprint 1 ","description":" first sprint "}`)
	wantStatus(t, rec, http.StatusCreated)
	created := decode[planResponse](t, rec)
	if created.ID == "" || created.ProjectID != project.ID || created.Name != "Sprint 1" ||
		created.Description != "first sprint" || created.TaskCount != 0 || created.TeamID != nil || created.TeamName != nil {
		t.Errorf("created = %+v", created)
	}

	list := decode[[]planResponse](t, env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans", ""))
	if len(list) != 2 || list[1].ID != created.ID {
		t.Errorf("plans = %+v, want [Main, Sprint 1]", list)
	}
}

func TestProjectCreatePlanErrors(t *testing.T) {
	tests := []struct {
		name      string
		projectID string
		body      string
		status    int
		code      string
	}{
		{"blank name", "", `{"name":"  "}`, http.StatusBadRequest, "invalid_input"},
		{"missing name", "", `{"description":"x"}`, http.StatusBadRequest, "invalid_input"},
		{"malformed json", "", `{"name":`, http.StatusBadRequest, "invalid_input"},
		{"unknown field", "", `{"name":"x","team_id":"t"}`, http.StatusBadRequest, "invalid_input"},
		{"duplicate name", "", `{"name":"Main"}`, http.StatusConflict, "conflict"},
		{"unknown project", "missing", `{"name":"x"}`, http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			project := createProject(t, env, "Alpha")
			id := tt.projectID
			if id == "" {
				id = project.ID
			}
			wantErrorCode(t, env.do(t, http.MethodPost, "/api/projects/"+id+"/plans", tt.body), tt.status, tt.code)

			list := decode[[]planResponse](t, env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans", ""))
			if len(list) != 1 {
				t.Errorf("plans = %+v, want only the default plan after a failed create", list)
			}
		})
	}
}
