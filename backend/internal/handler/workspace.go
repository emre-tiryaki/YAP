package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type WorkspaceHandler struct {
	session model.Session
	svc     *service.WorkspaceService
}

func NewWorkspaceHandler(session model.Session, svc *service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{session: session, svc: svc}
}

func (h *WorkspaceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/workspace", h.get)
	mux.HandleFunc("PATCH /api/workspace", h.update)
}

type workspaceResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserName string `json:"user_name"`
}

type updateWorkspaceRequest struct {
	Name     *string `json:"name"`
	UserName *string `json:"user_name"`
}

func toWorkspaceResponse(p model.Profile) workspaceResponse {
	return workspaceResponse{ID: p.WorkspaceID, Name: p.WorkspaceName, UserName: p.DisplayName}
}

func (h *WorkspaceHandler) get(w http.ResponseWriter, r *http.Request) {
	profile, err := h.svc.Get(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkspaceResponse(profile))
}

func (h *WorkspaceHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateWorkspaceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	profile, err := h.svc.Update(r.Context(), h.session, service.UpdateProfileInput{
		DisplayName:   req.UserName,
		WorkspaceName: req.Name,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkspaceResponse(profile))
}
