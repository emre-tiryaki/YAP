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
	mux.HandleFunc("GET /api/projects/{id}", h.get)
	mux.HandleFunc("GET /api/projects/{id}/plans", h.plans)
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

type planResponse struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	TeamID      *string `json:"team_id"`
	TeamName    *string `json:"team_name"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	TaskCount   int     `json:"task_count"`
}

func toPlanResponse(p model.Plan) planResponse {
	res := planResponse{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		TaskCount:   p.TaskCount,
	}
	if p.TeamID != "" {
		res.TeamID = &p.TeamID
		res.TeamName = &p.TeamName
	}
	return res
}

func(h *ProjectHandler) get(w http.ResponseWriter, r *http.Request) {
	project, err := h.svc.Get(r.Context(), h.session.WorkspaceID, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return	
	}
	writeJSON(w, http.StatusOK, toProjectResponse(project))
}

func(h *ProjectHandler) plans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.Plans(r.Context(), h.session.WorkspaceID, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]planResponse, 0, len(plans))
	for _, p := range plans {
		res = append(res, toPlanResponse(p))
	}
	writeJSON(w, http. StatusOK, res)
}