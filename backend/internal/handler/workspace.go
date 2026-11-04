package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type WorkspaceHandler struct {
	session model.Session
}

func NewWorkspaceHandler(session model.Session) *WorkspaceHandler {
	return &WorkspaceHandler{session: session}
}

func (h *WorkspaceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/workspace", h.get)
}

type workspaceResponse struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserName string `json:"user_name"`
}

func (h *WorkspaceHandler) get(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, workspaceResponse{
		ID:       h.session.UserID,
		Name:     h.session.WorkspaceName,
		UserName: h.session.DisplayName,
	})
}
