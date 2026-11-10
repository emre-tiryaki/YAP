package handler

import (
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
	mux.HandleFunc("DELETE /api/tasks/{id}", h.delete)
}

type taskResponse struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	ProjectID      string  `json:"project_id"`
	ProjectName    string  `json:"project_name"`
	DueDate        *string `json:"due_date"`
	StatusCategory string  `json:"status_category"`
}

type createTaskRequest struct {
	Title       string  `json:"title"`
	Description string  `json:"description"`
	DueDate     *string `json:"due_date"`
	PlanID      string  `json:"plan_id"`
	StatusID    string  `json:"status_id"`
}

type planTaskResponse struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	StatusID       string  `json:"status_id"`
	StatusCategory string  `json:"status_category"`
	Position       string  `json:"position"`
	DueDate        *string `json:"due_date"`
}

type moveTaskRequest struct {
	StatusID string `json:"status_id"`
	Index    int    `json:"index"`
}

func formatDueDate(d *time.Time) *string {
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
		ProjectID:      t.ProjectID,
		ProjectName:    t.ProjectName,
		StatusCategory: string(t.StatusCategory),
		DueDate:        formatDueDate(t.DueDate),
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

	var due *time.Time
	if req.DueDate != nil && *req.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			writeError(w, r, fmt.Errorf("%w: due_date", service.ErrInvalidInput))
			return
		}
		due = &parsed
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
		res = append(res, planTaskResponse{
			ID:             t.ID,
			Title:          t.Title,
			Description:    t.Description,
			StatusID:       t.StatusID,
			StatusCategory: string(t.StatusCategory),
			Position:       t.Position,
			DueDate:        formatDueDate(t.DueDate),
		})
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
