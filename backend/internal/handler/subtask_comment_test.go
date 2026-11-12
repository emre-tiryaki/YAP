package handler

import (
	"net/http"
	"testing"
)

func TestSubtaskEndpoints(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	parent := b.createTask(t, env, `{"title":"parent"}`)

	rec := env.do(t, http.MethodPost, "/api/tasks/"+parent.ID+"/subtasks", `{"title":" child "}`)
	wantStatus(t, rec, http.StatusCreated)
	sub := decode[subtaskResponse](t, rec)
	if sub.Title != "child" || sub.ParentID != parent.ID || sub.PlanID != b.plan.ID || sub.StatusCategory != "todo" {
		t.Errorf("POST subtasks = %+v", sub)
	}

	rec = env.do(t, http.MethodPost, "/api/tasks/"+sub.ID+"/done", `{"done":true}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[subtaskResponse](t, rec); got.CompletedAt == nil || got.StatusCategory != "todo" || got.StatusID != sub.StatusID {
		t.Errorf("POST done = %+v, want completed and left in its status", got)
	}

	rec = env.do(t, http.MethodGet, "/api/tasks/"+parent.ID+"/subtasks", "")
	wantStatus(t, rec, http.StatusOK)
	if list := decode[[]subtaskResponse](t, rec); len(list) != 1 || list[0].ID != sub.ID || list[0].CompletedAt == nil {
		t.Errorf("GET subtasks = %+v", list)
	}

	board := decode[[]planTaskResponse](t, env.do(t, http.MethodGet, b.path("/tasks"), ""))
	if len(board) != 1 || board[0].ID != parent.ID || board[0].SubtaskCount != 1 || board[0].DoneSubtaskCount != 1 {
		t.Errorf("board = %+v, want only the parent with 1/1 subtasks", board)
	}
}

func TestSubtaskEndpointErrors(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	parent := b.createTask(t, env, `{"title":"parent"}`)
	rec := env.do(t, http.MethodPost, "/api/tasks/"+parent.ID+"/subtasks", `{"title":"child"}`)
	wantStatus(t, rec, http.StatusCreated)
	child := decode[subtaskResponse](t, rec)
	missing := "00000000-0000-7000-8000-000000000999"

	tests := []struct {
		name, method, path, body string
		status                   int
		code                     string
	}{
		{"blank title", http.MethodPost, "/api/tasks/" + parent.ID + "/subtasks", `{"title":""}`, http.StatusBadRequest, "invalid_input"},
		{"nested", http.MethodPost, "/api/tasks/" + child.ID + "/subtasks", `{"title":"x"}`, http.StatusBadRequest, "invalid_input"},
		{"unknown parent", http.MethodPost, "/api/tasks/" + missing + "/subtasks", `{"title":"x"}`, http.StatusNotFound, "not_found"},
		{"list unknown parent", http.MethodGet, "/api/tasks/" + missing + "/subtasks", "", http.StatusNotFound, "not_found"},
		{"done unknown task", http.MethodPost, "/api/tasks/" + missing + "/done", `{"done":true}`, http.StatusNotFound, "not_found"},
		{"done malformed", http.MethodPost, "/api/tasks/" + child.ID + "/done", `{"done":"yes"}`, http.StatusBadRequest, "invalid_input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, tt.method, tt.path, tt.body), tt.status, tt.code)
		})
	}
}

func TestCommentEndpoints(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	task := b.createTask(t, env, `{"title":"task"}`)
	path := "/api/tasks/" + task.ID + "/comments"

	rec := env.do(t, http.MethodPost, path, `{"body":" hello "}`)
	wantStatus(t, rec, http.StatusCreated)
	c := decode[commentResponse](t, rec)
	if c.Body != "hello" || c.TaskID != task.ID || c.AuthorID != env.session.MembershipID ||
		c.AuthorName != "tester" || !c.IsOwn || c.CreatedAt == "" {
		t.Errorf("POST comments = %+v", c)
	}

	rec = env.do(t, http.MethodPatch, "/api/comments/"+c.ID, `{"body":"edited"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[commentResponse](t, rec); got.Body != "edited" || got.ID != c.ID {
		t.Errorf("PATCH comment = %+v", got)
	}

	rec = env.do(t, http.MethodGet, path, "")
	wantStatus(t, rec, http.StatusOK)
	if list := decode[[]commentResponse](t, rec); len(list) != 1 || list[0].Body != "edited" {
		t.Errorf("GET comments = %+v", list)
	}
	board := decode[[]planTaskResponse](t, env.do(t, http.MethodGet, b.path("/tasks"), ""))
	if len(board) != 1 || board[0].CommentCount != 1 {
		t.Errorf("board = %+v, want comment_count 1", board)
	}

	wantStatus(t, env.do(t, http.MethodDelete, "/api/comments/"+c.ID, ""), http.StatusNoContent)
	if list := decode[[]commentResponse](t, env.do(t, http.MethodGet, path, "")); len(list) != 0 {
		t.Errorf("GET comments after delete = %+v, want none", list)
	}
}

func TestCommentEndpointErrors(t *testing.T) {
	env := newTestEnv(t)
	b := newPlanBoard(t, env)
	task := b.createTask(t, env, `{"title":"task"}`)
	missing := "00000000-0000-7000-8000-000000000999"

	tests := []struct {
		name, method, path, body string
		status                   int
		code                     string
	}{
		{"blank body", http.MethodPost, "/api/tasks/" + task.ID + "/comments", `{"body":"  "}`, http.StatusBadRequest, "invalid_input"},
		{"unknown field", http.MethodPost, "/api/tasks/" + task.ID + "/comments", `{"text":"x"}`, http.StatusBadRequest, "invalid_input"},
		{"unknown task", http.MethodPost, "/api/tasks/" + missing + "/comments", `{"body":"x"}`, http.StatusNotFound, "not_found"},
		{"list unknown task", http.MethodGet, "/api/tasks/" + missing + "/comments", "", http.StatusNotFound, "not_found"},
		{"edit unknown", http.MethodPatch, "/api/comments/" + missing, `{"body":"x"}`, http.StatusNotFound, "not_found"},
		{"delete unknown", http.MethodDelete, "/api/comments/" + missing, "", http.StatusNotFound, "not_found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wantErrorCode(t, env.do(t, tt.method, tt.path, tt.body), tt.status, tt.code)
		})
	}
}
