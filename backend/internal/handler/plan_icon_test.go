package handler

import (
	"net/http"
	"testing"
)

func TestPlanIconRoundTrip(t *testing.T) {
	env := newTestEnv(t)
	projectID := createProject(t, env, "Alpha").ID
	base := "/api/projects/" + projectID + "/plans"

	rec := env.do(t, http.MethodPost, base, `{"name":"Sprint","icon":"🚀"}`)
	wantStatus(t, rec, http.StatusCreated)
	plan := decode[planResponse](t, rec)
	if plan.Icon != "🚀" {
		t.Fatalf("POST icon = %q, want 🚀", plan.Icon)
	}

	rec = env.do(t, http.MethodPatch, base+"/"+plan.ID, `{"icon":"📚"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[planResponse](t, rec); got.Icon != "📚" || got.Name != "Sprint" {
		t.Errorf("PATCH = %+v, want icon 📚 and name kept", got)
	}

	list := decode[[]planResponse](t, env.do(t, http.MethodGet, base, ""))
	found := false
	for _, p := range list {
		if p.ID == plan.ID {
			found = p.Icon == "📚"
		}
	}
	if !found {
		t.Errorf("GET list = %+v, want plan with icon 📚", list)
	}

	rec = env.do(t, http.MethodPatch, base+"/"+plan.ID, `{"icon":""}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[planResponse](t, env.do(t, http.MethodGet, base+"/"+plan.ID, "")); got.Icon != "" {
		t.Errorf("GET icon = %q, want empty", got.Icon)
	}
}

func TestPlanIconInvalid(t *testing.T) {
	env := newTestEnv(t)
	projectID := createProject(t, env, "Alpha").ID
	base := "/api/projects/" + projectID + "/plans"
	wantErrorCode(t, env.do(t, http.MethodPost, base, `{"name":"Sprint","icon":"abc"}`), http.StatusBadRequest, "invalid_input")
	plans := decode[[]planResponse](t, env.do(t, http.MethodGet, base, ""))
	wantErrorCode(t, env.do(t, http.MethodPatch, base+"/"+plans[0].ID, `{"icon":"abc"}`), http.StatusBadRequest, "invalid_input")
}
