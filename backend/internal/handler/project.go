package handler

import (
	"fmt"
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
	mux.HandleFunc("PATCH /api/projects/{id}", h.update)
	mux.HandleFunc("DELETE /api/projects/{id}", h.delete)
	mux.HandleFunc("GET /api/projects/{id}/plans", h.plans)
	mux.HandleFunc("POST /api/projects/{id}/plans", h.createPlan)
	mux.HandleFunc("GET /api/projects/{id}/plans/{planId}", h.getPlan)
	mux.HandleFunc("PATCH /api/projects/{id}/plans/{planId}", h.updatePlan)
	mux.HandleFunc("DELETE /api/projects/{id}/plans/{planId}", h.deletePlan)
	mux.HandleFunc("GET /api/projects/{id}/plans/{planId}/statuses", h.statuses)
	mux.HandleFunc("POST /api/projects/{id}/plans/{planId}/statuses", h.createStatus)
	mux.HandleFunc("PATCH /api/projects/{id}/plans/{planId}/statuses/{statusId}", h.updateStatus)
	mux.HandleFunc("DELETE /api/projects/{id}/plans/{planId}/statuses/{statusId}", h.deleteStatus)
	mux.HandleFunc("POST /api/projects/{id}/plans/{planId}/statuses/{statusId}/move", h.moveStatus)
}

type projectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	TaskCount   int    `json:"task_count"`
}

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func toProjectResponse(p model.Project) projectResponse {
	return projectResponse{ID: p.ID, Name: p.Name, Description: p.Description, Icon: p.Icon, TaskCount: p.TaskCount}
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
		Icon:        req.Icon,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toProjectResponse(project))
}

type createPlanRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type planResponse struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	TeamID      *string `json:"team_id"`
	TeamName    *string `json:"team_name"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	TaskCount   int     `json:"task_count"`
	DoneCount   int     `json:"done_task_count"`
}

func toPlanResponse(p model.Plan) planResponse {
	res := planResponse{
		ID:          p.ID,
		ProjectID:   p.ProjectID,
		Name:        p.Name,
		Description: p.Description,
		TaskCount:   p.TaskCount,
		DoneCount:   p.DoneCount,
	}
	if p.TeamID != "" {
		res.TeamID = &p.TeamID
		res.TeamName = &p.TeamName
	}
	return res
}

func (h *ProjectHandler) get(w http.ResponseWriter, r *http.Request) {
	project, err := h.svc.Get(r.Context(), h.session.WorkspaceID, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(project))
}

func (h *ProjectHandler) plans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.svc.Plans(r.Context(), h.session.WorkspaceID, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]planResponse, 0, len(plans))
	for _, p := range plans {
		res = append(res, toPlanResponse(p))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ProjectHandler) createPlan(w http.ResponseWriter, r *http.Request) {
	var req createPlanRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	plan, err := h.svc.CreatePlan(r.Context(), h.session.WorkspaceID, r.PathValue("id"), service.CreatePlanInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPlanResponse(plan))
}

type statusResponse struct {
	ID          string `json:"id"`
	PlanID      string `json:"plan_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Category    string `json:"category"`
	Position    string `json:"position"`
}

type createStatusRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Category    string `json:"category"`
}

func toStatusResponse(s model.Status) statusResponse {
	return statusResponse{
		ID:          s.ID,
		PlanID:      s.PlanID,
		Title:       s.Title,
		Description: s.Description,
		Color:       s.Color,
		Category:    string(s.Category),
		Position:    s.Position,
	}
}

func (h *ProjectHandler) getPlan(w http.ResponseWriter, r *http.Request) {
	plan, err := h.svc.GetPlan(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlanResponse(plan))
}

func (h *ProjectHandler) statuses(w http.ResponseWriter, r *http.Request) {
	statuses, err := h.svc.Statuses(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]statusResponse, 0, len(statuses))
	for _, s := range statuses {
		res = append(res, toStatusResponse(s))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ProjectHandler) createStatus(w http.ResponseWriter, r *http.Request) {
	var req createStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	category, err := parseStatusCategory(req.Category)
	if err != nil {
		writeError(w, r, err)
		return
	}
	status, err := h.svc.CreateStatus(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId"), service.CreateStatusInput{
		Title:       req.Title,
		Description: req.Description,
		Color:       req.Color,
		Category:    category,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toStatusResponse(status))
}

// updateStatusRequest is a partial update: omitted (or null) fields are kept.
type updateStatusRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Color       *string `json:"color"`
}

func (h *ProjectHandler) updateStatus(w http.ResponseWriter, r *http.Request) {
	var req updateStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	status, err := h.svc.UpdateStatus(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId"), r.PathValue("statusId"), service.UpdateStatusInput{
		Title:       req.Title,
		Description: req.Description,
		Color:       req.Color,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toStatusResponse(status))
}

func (h *ProjectHandler) deleteStatus(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteStatus(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId"), r.PathValue("statusId")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type moveStatusRequest struct {
	Index int `json:"index"`
}

func (h *ProjectHandler) moveStatus(w http.ResponseWriter, r *http.Request) {
	var req moveStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	err := h.svc.MoveStatus(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId"), r.PathValue("statusId"), req.Index)
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parseStatusCategory(v string) (model.StatusCategory, error) {
	switch c := model.StatusCategory(v); c {
	case model.StatusToDo, model.StatusInProgress, model.StatusDone:
		return c, nil
	}
	return "", fmt.Errorf("%w: category", service.ErrInvalidInput)
}

// updateProjectRequest is a partial update: omitted (or null) fields are kept.
// An empty icon removes it.
type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Icon        *string `json:"icon"`
}

func (h *ProjectHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateProjectRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	project, err := h.svc.Update(r.Context(), h.session.WorkspaceID, r.PathValue("id"), service.UpdateProjectInput{
		Name:        req.Name,
		Description: req.Description,
		Icon:        req.Icon,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(project))
}

func (h *ProjectHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), h.session.WorkspaceID, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// updatePlanRequest is a partial update: omitted (or null) fields are kept.
type updatePlanRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (h *ProjectHandler) updatePlan(w http.ResponseWriter, r *http.Request) {
	var req updatePlanRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	plan, err := h.svc.UpdatePlan(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId"), service.UpdatePlanInput{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPlanResponse(plan))
}

func (h *ProjectHandler) deletePlan(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeletePlan(r.Context(), h.session.WorkspaceID, r.PathValue("id"), r.PathValue("planId")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
