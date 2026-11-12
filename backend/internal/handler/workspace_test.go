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

func TestWorkspaceUpdate(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		wantStatus   int
		wantName     string
		wantUserName string
	}{
		{name: "both", body: `{"name":" Home ","user_name":"Ada"}`, wantStatus: http.StatusOK, wantName: "Home", wantUserName: "Ada"},
		{name: "user name only", body: `{"user_name":"Ada"}`, wantStatus: http.StatusOK, wantName: "Personal", wantUserName: "Ada"},
		{name: "empty", body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "blank name", body: `{"name":"  "}`, wantStatus: http.StatusBadRequest},
		{name: "unknown field", body: `{"nickname":"x"}`, wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newTestEnv(t)
			rec := env.do(t, http.MethodPatch, "/api/workspace", tt.body)
			if tt.wantStatus != http.StatusOK {
				wantErrorCode(t, rec, tt.wantStatus, "invalid_input")
				return
			}
			wantStatus(t, rec, http.StatusOK)
			got := decode[workspaceResponse](t, rec)
			if got.Name != tt.wantName || got.UserName != tt.wantUserName || got.ID != env.session.WorkspaceID {
				t.Errorf("response = %+v", got)
			}
			// GET reflects the change even though the session was bootstrapped earlier.
			if again := decode[workspaceResponse](t, env.do(t, http.MethodGet, "/api/workspace", "")); again != got {
				t.Errorf("GET after PATCH = %+v, want %+v", again, got)
			}
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
