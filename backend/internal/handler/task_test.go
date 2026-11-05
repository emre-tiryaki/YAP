package handler

import (
	"net/http"
	"strings"
	"testing"
)

func createProject(t *testing.T, env testEnv, name string) projectResponse {
	t.Helper()
	rec := env.do(t, http.MethodPost, "/api/projects", `{"name":"`+name+`"}`)
	wantStatus(t, rec, http.StatusCreated)
	return decode[projectResponse](t, rec)
}

func TestTaskUpcomingEmpty(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(t, http.MethodGet, "/api/tasks/upcoming", "")
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestTaskCreate(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantDue *string
	}{
		{"with due date", `{"title":" Write tests ","description":"d","due_date":"2026-12-01T10:00:00Z"}`, ptr("2026-12-01T10:00:00Z")},
		{"offset due date normalised to UTC", `{"title":"Write tests","due_date":"2026-12-01T13:00:00+03:00"}`, ptr("2026-12-01T10:00:00Z")},
		{"null due date", `{"title":"Write tests","due_date":null}`, nil},
		{"empty due date", `{"title":"Write tests","due_date":""}`, nil},
		{"no due date", `{"title":"Write tests"}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			project := createProject(t, env, "Backend")

			rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", tt.body)
			wantStatus(t, rec, http.StatusCreated)

			got := decode[taskResponse](t, rec)
			if got.ID == "" || got.Title != "Write tests" || got.ProjectID != project.ID {
				t.Errorf("task = %+v", got)
			}
			if got.ProjectName != "Backend" {
				t.Errorf("ProjectName = %q, want Backend", got.ProjectName)
			}
			if got.StatusCategory != "todo" {
				t.Errorf("StatusCategory = %q, want todo", got.StatusCategory)
			}
			switch {
			case tt.wantDue == nil && got.DueDate != nil:
				t.Errorf("DueDate = %q, want null", *got.DueDate)
			case tt.wantDue != nil && (got.DueDate == nil || *got.DueDate != *tt.wantDue):
				t.Errorf("DueDate = %v, want %q", got.DueDate, *tt.wantDue)
			}
		})
	}
}

func TestTaskCreateErrors(t *testing.T) {
	tests := []struct {
		name       string
		useProject bool
		projectID  string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"blank title", true, "", `{"title":" "}`, http.StatusBadRequest, "invalid_input"},
		{"malformed json", true, "", `{"title":`, http.StatusBadRequest, "invalid_input"},
		{"unknown field", true, "", `{"title":"x","extra":1}`, http.StatusBadRequest, "invalid_input"},
		{"invalid due date", true, "", `{"title":"x","due_date":"tomorrow"}`, http.StatusBadRequest, "invalid_input"},
		{"unknown project", false, "missing", `{"title":"x"}`, http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			projectID := tt.projectID
			if tt.useProject {
				projectID = createProject(t, env, "Backend").ID
			}
			rec := env.do(t, http.MethodPost, "/api/projects/"+projectID+"/tasks", tt.body)
			wantErrorCode(t, rec, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestTaskUpcomingOrderingAndShape(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Alpha")
	path := "/api/projects/" + project.ID + "/tasks"

	bodies := []string{
		`{"title":"later","due_date":"2026-12-03T09:00:00Z"}`,
		`{"title":"undated"}`,
		`{"title":"sooner","due_date":"2026-12-01T09:00:00Z"}`,
	}
	for _, b := range bodies {
		wantStatus(t, env.do(t, http.MethodPost, path, b), http.StatusCreated)
	}

	rec := env.do(t, http.MethodGet, "/api/tasks/upcoming", "")
	wantStatus(t, rec, http.StatusOK)
	got := decode[[]taskResponse](t, rec)

	if len(got) != 2 || got[0].Title != "sooner" || got[1].Title != "later" {
		t.Fatalf("upcoming = %+v, want [sooner, later]", got)
	}
	if got[0].ProjectName != "Alpha" || got[0].ProjectID != project.ID {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[0].DueDate == nil || *got[0].DueDate != "2026-12-01T09:00:00Z" {
		t.Errorf("DueDate = %v", got[0].DueDate)
	}
	if got[0].StatusCategory != "todo" {
		t.Errorf("StatusCategory = %q", got[0].StatusCategory)
	}
}

func TestTaskMethodNotAllowed(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Alpha")

	wantStatus(t, env.do(t, http.MethodPost, "/api/tasks/upcoming", ""), http.StatusMethodNotAllowed)
	wantStatus(t, env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/tasks", ""), http.StatusMethodNotAllowed)
}

func ptr(s string) *string { return &s }