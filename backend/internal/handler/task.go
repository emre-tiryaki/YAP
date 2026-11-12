package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type TaskHandler struct {
	session model.Session
	svc     *service.TaskService
}

func NewTaskHandler(session model.Session, svc *service.TaskService) *TaskHandler {
	return &TaskHandler{session: session, svc: svc}
}

func (h *TaskHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tasks/upcoming", h.upcoming)
	mux.HandleFunc("POST /api/projects/{id}/tasks", h.create)
	mux.HandleFunc("GET /api/projects/{id}/plans/{planId}/tasks", h.listByPlan)
	mux.HandleFunc("POST /api/tasks/{id}/move", h.move)
	mux.HandleFunc("PATCH /api/tasks/{id}", h.update)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.delete)
	mux.HandleFunc("GET /api/tasks/{id}/subtasks", h.listSubtasks)
	mux.HandleFunc("POST /api/tasks/{id}/subtasks", h.createSubtask)
	mux.HandleFunc("POST /api/tasks/{id}/done", h.setDone)
}

type taskResponse struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	ProjectID      string  `json:"project_id"`
	ProjectName    string  `json:"project_name"`
	PlanID         string  `json:"plan_id"`
	StatusID       string  `json:"status_id"`
	Position       string  `json:"position"`
	DueDate        *string `json:"due_date"`
	StatusCategory string  `json:"status_category"`
	CompletedAt    *string `json:"completed_at"`
}

type createTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DueDate     *string `json:"due_date"`
	PlanID      string  `json:"plan_id"`
	StatusID    string  `json:"status_id"`
}

type planTaskResponse struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	StatusID         string  `json:"status_id"`
	StatusCategory   string  `json:"status_category"`
	Position         string  `json:"position"`
	DueDate          *string `json:"due_date"`
	CompletedAt      *string `json:"completed_at"`
	SubtaskCount     int     `json:"subtask_count"`
	DoneSubtaskCount int     `json:"done_subtask_count"`
	CommentCount     int     `json:"comment_count"`
}

type subtaskResponse struct {
	ID             string  `json:"id"`
	ParentID       string  `json:"parent_id"`
	Title          string  `json:"title"`
	PlanID         string  `json:"plan_id"`
	StatusID       string  `json:"status_id"`
	StatusCategory string  `json:"status_category"`
	CompletedAt    *string `json:"completed_at"`
}

type createSubtaskRequest struct {
	Title string `json:"title"`
}

type setDoneRequest struct {
	Done bool `json:"done"`
}

type moveTaskRequest struct {
	StatusID string `json:"status_id"`
	Index    int    `json:"index"`
}

func formatTime(d *time.Time) *string {
	if d == nil {
		return nil
	}
	s := d.UTC().Format(time.RFC3339)
	return &s
}

func toTaskResponse(t model.Task) taskResponse {
	return taskResponse{
		ID:             t.ID,
		Title:          t.Title,
		Description:    t.Description,
		ProjectID:      t.ProjectID,
		ProjectName:    t.ProjectName,
		PlanID:         t.PlanID,
		StatusID:       t.StatusID,
		Position:       t.Position,
		StatusCategory: string(t.StatusCategory),
		DueDate:        formatTime(t.DueDate),
		CompletedAt:    formatTime(t.CompletedAt),
	}
}

func toPlanTaskResponse(t model.Task) planTaskResponse {
	return planTaskResponse{
		ID:               t.ID,
		Title:            t.Title,
		Description:      t.Description,
		StatusID:         t.StatusID,
		StatusCategory:   string(t.StatusCategory),
		Position:         t.Position,
		DueDate:          formatTime(t.DueDate),
		CompletedAt:      formatTime(t.CompletedAt),
		SubtaskCount:     t.SubtaskCount,
		DoneSubtaskCount: t.DoneSubtaskCount,
		CommentCount:     t.CommentCount,
	}
}

func toSubtaskResponse(t model.Task) subtaskResponse {
	return subtaskResponse{
		ID:             t.ID,
		ParentID:       t.ParentID,
		Title:          t.Title,
		PlanID:         t.PlanID,
		StatusID:       t.StatusID,
		StatusCategory: string(t.StatusCategory),
		CompletedAt:    formatTime(t.CompletedAt),
	}
}

func (h *TaskHandler) upcoming(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.Upcoming(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]taskResponse, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, toTaskResponse(t))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *TaskHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	var raw string
	if req.DueDate != nil {
		raw = *req.DueDate
	}
	due, err := parseDueDate(raw)
	if err != nil {
		writeError(w, r, err)
		return
	}

	task, err := h.svc.Create(r.Context(), h.session, r.PathValue("id"), service.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		DueDate:     due,
		PlanID:      req.PlanID,
		StatusID:    req.StatusID,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTaskResponse(task))
}

func (h *TaskHandler) listByPlan(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListByPlan(r.Context(), h.session, r.PathValue("id"), r.PathValue("planId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]planTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, toPlanTaskResponse(t))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *TaskHandler) move(w http.ResponseWriter, r *http.Request) {
	var req moveTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	err := h.svc.Move(r.Context(), h.session, r.PathValue("id"), service.MoveTaskInput{
		StatusID: req.StatusID,
		Index:    req.Index,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// parseDueDate parses an RFC 3339 due date; an empty string means no date.
func parseDueDate(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, fmt.Errorf("%w: due_date", service.ErrInvalidInput)
	}
	return &parsed, nil
}

// updateTaskRequest is a partial update: omitted (or null) title and
// description are kept. due_date is changed only when present; null or ""
// clears it. It stays raw so a present null can be told apart from absence.
type updateTaskRequest struct {
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	DueDate     json.RawMessage `json:"due_date"`
}

func (h *TaskHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	in := service.UpdateTaskInput{Title: req.Title, Description: req.Description}
	if req.DueDate != nil {
		var raw *string
		if err := json.Unmarshal(req.DueDate, &raw); err != nil {
			writeError(w, r, fmt.Errorf("%w: due_date", service.ErrInvalidInput))
			return
		}
		var v string
		if raw != nil {
			v = *raw
		}
		due, err := parseDueDate(v)
		if err != nil {
			writeError(w, r, err)
			return
		}
		in.DueDate, in.DueDateSet = due, true
	}

	task, err := h.svc.Update(r.Context(), h.session, r.PathValue("id"), in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlanTaskResponse(task))
}

func (h *TaskHandler) listSubtasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.Subtasks(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]subtaskResponse, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, toSubtaskResponse(t))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *TaskHandler) createSubtask(w http.ResponseWriter, r *http.Request) {
	var req createSubtaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	task, err := h.svc.CreateSubtask(r.Context(), h.session, r.PathValue("id"), req.Title)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toSubtaskResponse(task))
}

func (h *TaskHandler) setDone(w http.ResponseWriter, r *http.Request) {
	var req setDoneRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	task, err := h.svc.SetDone(r.Context(), h.session, r.PathValue("id"), req.Done)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSubtaskResponse(task))
}
