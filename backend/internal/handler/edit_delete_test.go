package handler

import (
	"net/http"
	"reflect"
	"testing"
)

func TestProjectUpdate(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		wantName        string
		wantDescription string
	}{
		{"rename", `{"name":"  Beta "}`, "Beta", "first"},
		{"description", `{"description":" second "}`, "Alpha", "second"},
		{"both", `{"name":"Beta","description":"second"}`, "Beta", "second"},
		{"clear description", `{"description":""}`, "Alpha", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			rec := env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha","description":"first"}`)
			wantStatus(t, rec, http.StatusCreated)
			project := decode[projectResponse](t, rec)

			rec = env.do(t, http.MethodPatch, "/api/projects/"+project.ID, tt.body)
			wantStatus(t, rec, http.StatusOK)
			got := decode[projectResponse](t, rec)
			if got.ID != project.ID || got.Name != tt.wantName || got.Description != tt.wantDescription {
				t.Errorf("PATCH = %+v, want name %q description %q", got, tt.wantName, tt.wantDescription)
			}

			stored := decode[projectResponse](t, env.do(t, http.MethodGet, "/api/projects/"+project.ID, ""))
			if stored.Name != tt.wantName || stored.Description != tt.wantDescription {
				t.Errorf("GET = %+v", stored)
			}
		})
	}
}

func TestProjectUpdateErrors(t *testing.T) {
	tests := []struct {
		name       string
		missing    bool
		body       string
		wantStatus int
		wantCode   string
	}{
		{"blank name", false, `{"name":"  "}`, http.StatusBadRequest, "invalid_input"},
		{"null name", false, `{"name":null}`, http.StatusBadRequest, "invalid_input"},
		{"no fields", false, `{}`, http.StatusBadRequest, "invalid_input"},
		{"malformed json", false, `{"name":`, http.StatusBadRequest, "invalid_input"},
		{"unknown field", false, `{"name":"x","extra":1}`, http.StatusBadRequest, "invalid_input"},
		{"unknown project", true, `{"name":"x"}`, http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			id := "missing"
			if !tt.missing {
				id = createProject(t, env, "Alpha").ID
			}
			wantErrorCode(t, env.do(t, http.MethodPatch, "/api/projects/"+id, tt.body), tt.wantStatus, tt.wantCode)
		})
	}
}

func TestProjectDelete(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Alpha")
	other := createProject(t, env, "Other")
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"t","due_date":"2026-12-01T10:00:00Z"}`), http.StatusCreated)

	rec := env.do(t, http.MethodDelete, "/api/projects/"+project.ID, "")
	wantStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}

	wantErrorCode(t, env.do(t, http.MethodGet, "/api/projects/"+project.ID, ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans", ""), http.StatusNotFound, "not_found")
	list := decode[[]projectResponse](t, env.do(t, http.MethodGet, "/api/projects", ""))
	if len(list) != 1 || list[0].ID != other.ID {
		t.Errorf("projects = %+v, want only Other", list)
	}
	upcoming := decode[[]taskResponse](t, env.do(t, http.MethodGet, "/api/tasks/upcoming", ""))
	if len(upcoming) != 0 {
		t.Errorf("upcoming = %+v, want none", upcoming)
	}

	wantErrorCode(t, env.do(t, http.MethodDelete, "/api/projects/"+project.ID, ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, env.do(t, http.MethodDelete, "/api/projects/missing", ""), http.StatusNotFound, "not_found")
}

func TestPlanUpdate(t *testing.T) {
	env := newTestEnv(t)
	board := newPlanBoard(t, env)
	board.createTask(t, env, `{"title":"t"}`)

	rec := env.do(t, http.MethodPatch, board.path(""), `{"name":" Renamed ","description":" about "}`)
	wantStatus(t, rec, http.StatusOK)
	got := decode[planResponse](t, rec)
	if got.ID != board.plan.ID || got.ProjectID != board.project.ID || got.Name != "Renamed" ||
		got.Description != "about" || got.TaskCount != 1 || got.TeamID != nil {
		t.Errorf("PATCH = %+v", got)
	}

	rec = env.do(t, http.MethodPatch, board.path(""), `{"description":"only"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[planResponse](t, rec); got.Name != "Renamed" || got.Description != "only" {
		t.Errorf("PATCH description = %+v", got)
	}

	stored := decode[planResponse](t, env.do(t, http.MethodGet, board.path(""), ""))
	if stored.Name != "Renamed" || stored.Description != "only" {
		t.Errorf("GET = %+v", stored)
	}
}

func TestPlanUpdateErrors(t *testing.T) {
	env := newTestEnv(t)
	board := newPlanBoard(t, env)
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects/"+board.project.ID+"/plans", `{"name":"Taken"}`), http.StatusCreated)
	other := newPlanBoard(t, env)

	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"blank name", board.path(""), `{"name":" "}`, http.StatusBadRequest, "invalid_input"},
		{"no fields", board.path(""), `{}`, http.StatusBadRequest, "invalid_input"},
		{"malformed json", board.path(""), `{`, http.StatusBadRequest, "invalid_input"},
		{"unknown field", board.path(""), `{"team_id":"x"}`, http.StatusBadRequest, "invalid_input"},
		{"duplicate name", board.path(""), `{"name":"Taken"}`, http.StatusConflict, "conflict"},
		{"unknown plan", "/api/projects/" + board.project.ID + "/plans/missing", `{"name":"x"}`, http.StatusNotFound, "not_found"},
		{"plan of another project", "/api/projects/" + board.project.ID + "/plans/" + other.plan.ID, `{"name":"x"}`, http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodPatch, tt.path, tt.body), tt.wantStatus, tt.wantCode)
		})
	}
}

func TestPlanDelete(t *testing.T) {
	env := newTestEnv(t)
	board := newPlanBoard(t, env)
	rec := env.do(t, http.MethodPost, "/api/projects/"+board.project.ID+"/plans", `{"name":"Sprint"}`)
	wantStatus(t, rec, http.StatusCreated)
	sprint := decode[planResponse](t, rec)
	sprintPath := "/api/projects/" + board.project.ID + "/plans/" + sprint.ID
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects/"+board.project.ID+"/tasks", `{"title":"t","plan_id":"`+sprint.ID+`"}`), http.StatusCreated)

	rec = env.do(t, http.MethodDelete, sprintPath, "")
	wantStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}

	wantErrorCode(t, env.do(t, http.MethodGet, sprintPath, ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, env.do(t, http.MethodGet, sprintPath+"/tasks", ""), http.StatusNotFound, "not_found")
	plans := decode[[]planResponse](t, env.do(t, http.MethodGet, "/api/projects/"+board.project.ID+"/plans", ""))
	if len(plans) != 1 || plans[0].ID != board.plan.ID {
		t.Errorf("plans = %+v, want only Main", plans)
	}
	project := decode[projectResponse](t, env.do(t, http.MethodGet, "/api/projects/"+board.project.ID, ""))
	if project.TaskCount != 0 {
		t.Errorf("project TaskCount = %d, want 0", project.TaskCount)
	}

	wantErrorCode(t, env.do(t, http.MethodDelete, sprintPath, ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, env.do(t, http.MethodDelete, board.path(""), ""), http.StatusConflict, "conflict")
	wantErrorCode(t, env.do(t, http.MethodDelete, "/api/projects/missing/plans/"+board.plan.ID, ""), http.StatusNotFound, "not_found")
}

func TestTaskUpdate(t *testing.T) {
	tests := []struct {
		name            string
		body            string
		wantTitle       string
		wantDescription string
		wantDue         *string
	}{
		{"rename keeps the rest", `{"title":" Renamed "}`, "Renamed", "desc", ptr("2026-12-01T10:00:00Z")},
		{"description", `{"description":"other"}`, "Write", "other", ptr("2026-12-01T10:00:00Z")},
		{"due date normalised to UTC", `{"due_date":"2027-01-02T13:00:00+03:00"}`, "Write", "desc", ptr("2027-01-02T10:00:00Z")},
		{"null due date clears", `{"due_date":null}`, "Write", "desc", nil},
		{"empty due date clears", `{"due_date":""}`, "Write", "desc", nil},
		{"all fields", `{"title":"A","description":"","due_date":"2027-01-02T10:00:00Z"}`, "A", "", ptr("2027-01-02T10:00:00Z")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			board := newPlanBoard(t, env)
			created := board.createTask(t, env, `{"title":"Write","description":"desc","due_date":"2026-12-01T10:00:00Z"}`)

			rec := env.do(t, http.MethodPatch, "/api/tasks/"+created.ID, tt.body)
			wantStatus(t, rec, http.StatusOK)
			got := decode[planTaskResponse](t, rec)
			if got.ID != created.ID || got.Title != tt.wantTitle || got.Description != tt.wantDescription {
				t.Errorf("PATCH = %+v, want title %q description %q", got, tt.wantTitle, tt.wantDescription)
			}
			if got.StatusID != board.statuses[0].ID || got.StatusCategory != "todo" || got.Position == "" {
				t.Errorf("PATCH placement = %+v", got)
			}
			switch {
			case tt.wantDue == nil && got.DueDate != nil:
				t.Errorf("DueDate = %q, want null", *got.DueDate)
			case tt.wantDue != nil && (got.DueDate == nil || *got.DueDate != *tt.wantDue):
				t.Errorf("DueDate = %v, want %q", got.DueDate, *tt.wantDue)
			}

			listed := decode[[]planTaskResponse](t, env.do(t, http.MethodGet, board.path("/tasks"), ""))
			if len(listed) != 1 || !reflect.DeepEqual(listed[0], got) {
				t.Errorf("listed = %+v, want [%+v]", listed, got)
			}
		})
	}
}

func TestTaskUpdateErrors(t *testing.T) {
	env := newTestEnv(t)
	board := newPlanBoard(t, env)
	task := board.createTask(t, env, `{"title":"x"}`)
	deleted := board.createTask(t, env, `{"title":"gone"}`)
	wantStatus(t, env.do(t, http.MethodDelete, "/api/tasks/"+deleted.ID, ""), http.StatusNoContent)

	tests := []struct {
		name       string
		id         string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"blank title", task.ID, `{"title":" "}`, http.StatusBadRequest, "invalid_input"},
		{"null title", task.ID, `{"title":null}`, http.StatusBadRequest, "invalid_input"},
		{"no fields", task.ID, `{}`, http.StatusBadRequest, "invalid_input"},
		{"invalid due date", task.ID, `{"due_date":"tomorrow"}`, http.StatusBadRequest, "invalid_input"},
		{"due date not a string", task.ID, `{"due_date":5}`, http.StatusBadRequest, "invalid_input"},
		{"malformed json", task.ID, `{"title":`, http.StatusBadRequest, "invalid_input"},
		{"unknown field", task.ID, `{"status_id":"x"}`, http.StatusBadRequest, "invalid_input"},
		{"unknown task", "missing", `{"title":"x"}`, http.StatusNotFound, "not_found"},
		{"deleted task", deleted.ID, `{"title":"x"}`, http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, http.MethodPatch, "/api/tasks/"+tt.id, tt.body), tt.wantStatus, tt.wantCode)
		})
	}
}

func TestEditDeleteMethodNotAllowed(t *testing.T) {
	env := newTestEnv(t)
	board := newPlanBoard(t, env)
	task := board.createTask(t, env, `{"title":"x"}`)

	for _, tc := range []struct{ method, path string }{
		{http.MethodPut, "/api/projects/" + board.project.ID},
		{http.MethodPut, board.path("")},
		{http.MethodPost, board.path("")},
		{http.MethodPut, "/api/tasks/" + task.ID},
		{http.MethodGet, "/api/tasks/" + task.ID},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			wantStatus(t, env.do(t, tc.method, tc.path, ""), http.StatusMethodNotAllowed)
		})
	}
}
