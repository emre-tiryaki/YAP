package handler

import (
	"net/http"
	"testing"
)

func TestWorkspaceGet(t *testing.T) {
	env := newTestEnv(t)

	rec := env.do(t, http.MethodGet, "/api/workspace", "")
	wantStatus(t, rec, http.StatusOK)

	got := decode[workspaceResponse](t, rec)
	if got.ID != env.session.WorkspaceID {
		t.Errorf("ID = %q, want workspace id %q", got.ID, env.session.WorkspaceID)
	}
	if got.Name != env.session.WorkspaceName || got.UserName != "tester" {
		t.Errorf("response = %+v", got)
	}
}

func TestWorkspaceMethodNotAllowed(t *testing.T) {
	env := newTestEnv(t)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			wantStatus(t, env.do(t, method, "/api/workspace", ""), http.StatusMethodNotAllowed)
		})
	}
}

func TestHelloWorldRoute(t *testing.T) {
	env := newTestEnv(t)
	rec := env.do(t, http.MethodGet, "/hello-world", "")
	wantStatus(t, rec, http.StatusOK)
	if rec.Body.String() != "Server is working!!!" {
		t.Errorf("body = %q", rec.Body.String())
	}
}