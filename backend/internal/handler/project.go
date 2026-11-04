package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type ProjectHandler struct {
	session model.Session
	svc     *service.ProjectService
}

func NewProjectHandler(session model.Session, svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{session: session, svc: svc}
}

func (h *ProjectHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects", h.list)
	mux.HandleFunc("POST /api/projects", h.create)
}

type projectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TaskCount   int    `json:"task_count"`
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func toProjectResponse(p model.Project) projectResponse {
	return projectResponse{ID: p.ID, Name: p.Name, Description: p.Description, TaskCount: p.TaskCount}
}

func (h *ProjectHandler) list(w http.ResponseWriter, r *http.Request) {
	projects, err := h.svc.List(r.Context(), h.session.WorkspaceID)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]projectResponse, 0, len(projects))
	for _, p := range projects {
		res = append(res, toProjectResponse(p))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ProjectHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	project, err := h.svc.Create(r.Context(), h.session.WorkspaceID, service.CreateProjectInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectResponse(project))
}
