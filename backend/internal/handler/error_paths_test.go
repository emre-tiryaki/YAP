package handler

import (
	"net/http"
	"testing"
)

func TestErrorPathsOfThinHandlers(t *testing.T) {
	e := newTestEnv(t)
	project := decode[projectResponse](t, e.do(t, "POST", "/api/projects", `{"name":"P"}`))
	label := decode[labelResponse](t, e.do(t, "POST", "/api/projects/"+project.ID+"/labels", `{"name":"bug","color":"red"}`))
	other := decode[labelResponse](t, e.do(t, "POST", "/api/projects/"+project.ID+"/labels", `{"name":"docs","color":"blue"}`))

	tests := []struct {
		name         string
		method, path string
		body         string
		status       int
		code         string
	}{
		{"label update bad json", "PATCH", "/api/labels/" + label.ID, `{`, 400, "invalid_input"},
		{"label update missing", "PATCH", "/api/labels/nope", `{"name":"x","color":"red"}`, 404, "not_found"},
		{"label update bad color", "PATCH", "/api/labels/" + label.ID, `{"name":"x","color":"neon"}`, 400, "invalid_input"},
		{"label update taken name", "PATCH", "/api/labels/" + label.ID, `{"name":"docs","color":"red"}`, 409, "conflict"},
		{"label delete missing", "DELETE", "/api/labels/nope", "", 404, "not_found"},
		{"ai delete connection missing", "DELETE", "/api/ai/connections/nope", "", 404, "not_found"},
		{"ai delete chat missing", "DELETE", "/api/ai/chats/nope", "", 404, "not_found"},
		{"ai get chat missing", "GET", "/api/ai/chats/nope", "", 404, "not_found"},
		{"ai stop chat missing", "POST", "/api/ai/chats/nope/stop", "", 404, "not_found"},
		{"ai send to missing chat", "POST", "/api/ai/chats/nope/messages", `{"connection_id":"x","model":"m1","content":"hi"}`, 404, "not_found"},
		{"ai regenerate missing chat", "POST", "/api/ai/chats/nope/regenerate", `{"connection_id":"x","model":"m1"}`, 404, "not_found"},
		{"ai send bad json", "POST", "/api/ai/chats/nope/messages", `{`, 400, "invalid_input"},
		{"ai regenerate bad json", "POST", "/api/ai/chats/nope/regenerate", `{`, 400, "invalid_input"},
		{"task resources unknown task", "GET", "/api/tasks/nope/resources", "", 404, "not_found"},
		{"resource tasks unknown resource", "GET", "/api/resources/nope/tasks", "", 404, "not_found"},
		{"labels of unknown project", "GET", "/api/projects/nope/labels", "", 404, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, e.do(t, tt.method, tt.path, tt.body), tt.status, tt.code)
		})
	}

	// A rejected update changes nothing.
	list := decode[[]labelResponse](t, e.do(t, "GET", "/api/projects/"+project.ID+"/labels", ""))
	if len(list) != 2 || list[0].Name != "bug" || list[1].Name != other.Name {
		t.Fatalf("labels after rejected updates = %+v", list)
	}
	wantStatus(t, e.do(t, "DELETE", "/api/labels/"+label.ID, ""), http.StatusNoContent)
}

func TestAIDeleteConnectionAndChat(t *testing.T) {
	e := newTestEnv(t)
	conn := e.aiConnection(t)
	wantStatus(t, e.do(t, "DELETE", "/api/ai/connections/"+conn.ID, ""), http.StatusNoContent)
	wantErrorCode(t, e.do(t, "DELETE", "/api/ai/connections/"+conn.ID, ""), http.StatusNotFound, "not_found")
}
