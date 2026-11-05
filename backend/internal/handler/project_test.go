package handler

import (
	"net/http"
	"strings"
	"testing"
)

func TestProjectListEmpty(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(t, http.MethodGet, "/api/projects", "")
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestProjectCreateAndList(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/projects", `{"name":"  Alpha ","description":"first"}`)
	wantStatus(t, rec, http.StatusCreated)
	created := decode[projectResponse](t, rec)
	if created.ID == "" || created.Name != "Alpha" || created.Description != "first" || created.TaskCount != 0 {
		t.Errorf("created = %+v", created)
	}

	rec = env.do(t, http.MethodGet, "/api/projects", "")
	wantStatus(t, rec, http.StatusOK)
	list := decode[[]projectResponse](t, rec)
	if len(list) != 1 || list[0] != created {
		t.Errorf("list = %+v, want [%+v]", list, created)
	}
}

func TestProjectListReportsTaskCount(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha"}`)
	wantStatus(t, rec, http.StatusCreated)
	project := decode[projectResponse](t, rec)

	for _, title := range []string{"one", "two"} {
		rec = env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"`+title+`"}`)
		wantStatus(t, rec, http.StatusCreated)
	}

	list := decode[[]projectResponse](t, env.do(t, http.MethodGet, "/api/projects", ""))
	if len(list) != 1 || list[0].TaskCount != 2 {
		t.Errorf("list = %+v, want task_count 2", list)
	}
}

func TestProjectCreateErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"blank name", `{"name":"  "}`},
		{"missing name", `{"description":"x"}`},
		{"malformed json", `{"name":`},
		{"unknown field", `{"name":"x","extra":true}`},
		{"wrong type", `{"name":123}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			wantErrorCode(t, env.do(t, http.MethodPost, "/api/projects", tt.body), http.StatusBadRequest, "invalid_input")

			list := decode[[]projectResponse](t, env.do(t, http.MethodGet, "/api/projects", ""))
			if len(list) != 0 {
				t.Errorf("list = %+v, want no projects after a failed create", list)
			}
		})
	}
}

func TestProjectCreateBodyTooLarge(t *testing.T) {
	env := newTestEnv(t)
	body := `{"name":"` + strings.Repeat("a", maxBodyBytes+10) + `"}`
	wantErrorCode(t, env.do(t, http.MethodPost, "/api/projects", body), http.StatusBadRequest, "invalid_input")
}

func TestProjectMethodNotAllowed(t *testing.T) {
	env := newTestEnv(t)
	for _, method := range []string{http.MethodPut, http.MethodDelete, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			wantStatus(t, env.do(t, method, "/api/projects", ""), http.StatusMethodNotAllowed)
		})
	}
}