package handler

import (
	"net/http"
	"testing"
)

func TestResourceGetAndPurge(t *testing.T) {
	e := newTestEnv(t)
	file := decode[resourceResponse](t, e.raw(t, "POST", "/api/resources/upload?name=a.txt", "", "x"))

	rec := e.do(t, "GET", "/api/resources/"+file.ID, "")
	wantStatus(t, rec, http.StatusOK)
	if got := decode[resourceResponse](t, rec); got.ID != file.ID || got.Name != "a.txt" {
		t.Fatalf("get = %+v", got)
	}
	wantErrorCode(t, e.do(t, "GET", "/api/resources/missing", ""), http.StatusNotFound, "not_found")

	// Only trashed resources can be purged; a live one reads as not found.
	wantErrorCode(t, e.do(t, "DELETE", "/api/resources/"+file.ID, ""), http.StatusNotFound, "not_found")
	wantStatus(t, e.do(t, "POST", "/api/resources/"+file.ID+"/trash", ""), http.StatusNoContent)
	wantStatus(t, e.do(t, "DELETE", "/api/resources/"+file.ID, ""), http.StatusNoContent)
	wantErrorCode(t, e.do(t, "GET", "/api/resources/"+file.ID, ""), http.StatusNotFound, "not_found")
	wantErrorCode(t, e.do(t, "DELETE", "/api/resources/"+file.ID, ""), http.StatusNotFound, "not_found")
}

func TestResourceRouteErrors(t *testing.T) {
	e := newTestEnv(t)
	file := decode[resourceResponse](t, e.raw(t, "POST", "/api/resources/upload?name=a.txt", "", "x"))
	task := e.taskID(t)

	tests := []struct {
		name         string
		method, path string
		body         string
		status       int
		code         string
	}{
		{"create file bad json", "POST", "/api/resources/files", `{`, 400, "invalid_input"},
		{"create file blank name", "POST", "/api/resources/files", `{"name":"  "}`, 400, "invalid_input"},
		{"create file duplicate", "POST", "/api/resources/files", `{"name":"A.TXT"}`, 409, "conflict"},
		{"create file missing parent", "POST", "/api/resources/files", `{"parent_id":"nope","name":"x"}`, 404, "not_found"},
		{"rename bad json", "PATCH", "/api/resources/" + file.ID, `{`, 400, "invalid_input"},
		{"rename blank", "PATCH", "/api/resources/" + file.ID, `{"name":""}`, 400, "invalid_input"},
		{"rename missing", "PATCH", "/api/resources/nope", `{"name":"x"}`, 404, "not_found"},
		{"move bad json", "POST", "/api/resources/" + file.ID + "/move", `{`, 400, "invalid_input"},
		{"move missing", "POST", "/api/resources/nope/move", `{}`, 404, "not_found"},
		{"move into file", "POST", "/api/resources/" + file.ID + "/move", `{"parent_id":"` + file.ID + `"}`, 400, "invalid_input"},
		{"copy missing", "POST", "/api/resources/nope/copy", `{}`, 404, "not_found"},
		{"trash missing", "POST", "/api/resources/nope/trash", "", 404, "not_found"},
		{"restore not trashed", "POST", "/api/resources/" + file.ID + "/restore", "", 404, "not_found"},
		{"unlink missing resource", "DELETE", "/api/tasks/" + task + "/resources/nope", "", 404, "not_found"},
		{"unlink missing task", "DELETE", "/api/tasks/nope/resources/" + file.ID, "", 404, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, e.do(t, tt.method, tt.path, tt.body), tt.status, tt.code)
		})
	}
}

func (e testEnv) taskID(t *testing.T) string {
	t.Helper()
	project := decode[projectResponse](t, e.do(t, "POST", "/api/projects", `{"name":"P"}`))
	rec := e.do(t, "POST", "/api/projects/"+project.ID+"/tasks", `{"title":"T"}`)
	wantStatus(t, rec, http.StatusCreated)
	return decode[struct {
		ID string `json:"id"`
	}](t, rec).ID
}
