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
}

func toTaskResponse(t model.Task) taskResponse {
	res := taskResponse{
		ID:             t.ID,
		Title:          t.Title,
		ProjectID:      t.ProjectID,
		ProjectName:    t.ProjectName,
		StatusCategory: string(t.StatusCategory),
	}
	if t.DueDate != nil {
		due := t.DueDate.UTC().Format(time.RFC3339)
		res.DueDate = &due
	}
	return res
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
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toTaskResponse(task))
}
