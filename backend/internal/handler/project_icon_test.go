package handler

import (
	"net/http"
	"testing"
)

func TestProjectIconRoundTrip(t *testing.T) {
	env := newTestEnv(t)
	rec := env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha","icon":"🚀"}`)
	wantStatus(t, rec, http.StatusCreated)
	project := decode[projectResponse](t, rec)
	if project.Icon != "🚀" {
		t.Fatalf("POST icon = %q, want 🚀", project.Icon)
	}

	rec = env.do(t, http.MethodPatch, "/api/projects/"+project.ID, `{"icon":"📚"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[projectResponse](t, rec); got.Icon != "📚" || got.Name != "Alpha" {
		t.Errorf("PATCH = %+v, want icon 📚 and name kept", got)
	}

	list := decode[[]projectResponse](t, env.do(t, http.MethodGet, "/api/projects", ""))
	if len(list) != 1 || list[0].Icon != "📚" {
		t.Errorf("GET list = %+v", list)
	}

	rec = env.do(t, http.MethodPatch, "/api/projects/"+project.ID, `{"icon":""}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[projectResponse](t, env.do(t, http.MethodGet, "/api/projects/"+project.ID, "")); got.Icon != "" {
		t.Errorf("GET icon = %q, want empty", got.Icon)
	}
}

func TestProjectIconInvalid(t *testing.T) {
	env := newTestEnv(t)
	wantErrorCode(t, env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha","icon":"abc"}`), http.StatusBadRequest, "invalid_input")
	id := createProject(t, env, "Beta").ID
	wantErrorCode(t, env.do(t, http.MethodPatch, "/api/projects/"+id, `{"icon":"abc"}`), http.StatusBadRequest, "invalid_input")
}
