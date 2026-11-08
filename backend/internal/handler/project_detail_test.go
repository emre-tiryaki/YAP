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
			wantStatus(t, env.do(t, method, "/api/projects/"+project.ID+"/plans", ""), http.StatusMethodNotAllowed)
		})
	}
}