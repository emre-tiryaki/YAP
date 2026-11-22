package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type LabelHandler struct {
	session model.Session
	svc     *service.LabelService
}

func NewLabelHandler(session model.Session, svc *service.LabelService) *LabelHandler {
	return &LabelHandler{session: session, svc: svc}
}

func (h *LabelHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects/{id}/labels", h.list)
	mux.HandleFunc("POST /api/projects/{id}/labels", h.create)
	mux.HandleFunc("PATCH /api/labels/{id}", h.update)
	mux.HandleFunc("DELETE /api/labels/{id}", h.delete)
}

type labelResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"`
	Color     string `json:"color"`
	Position  string `json:"position"`
}

func toLabelResponse(l model.Label) labelResponse {
	return labelResponse{ID: l.ID, ProjectID: l.ProjectID, Name: l.Name, Color: l.Color, Position: l.Position}
}

type createLabelRequest struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// updateLabelRequest is a partial update: omitted fields are kept.
type updateLabelRequest struct {
	Name  *string `json:"name"`
	Color *string `json:"color"`
}

func (h *LabelHandler) list(w http.ResponseWriter, r *http.Request) {
	labels, err := h.svc.List(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]labelResponse, 0, len(labels))
	for _, l := range labels {
		res = append(res, toLabelResponse(l))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *LabelHandler) create(w http.ResponseWriter, r *http.Request) {
	var req createLabelRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	label, err := h.svc.Create(r.Context(), h.session, r.PathValue("id"), service.CreateLabelInput{Name: req.Name, Color: req.Color})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toLabelResponse(label))
}

func (h *LabelHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateLabelRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	label, err := h.svc.Update(r.Context(), h.session, r.PathValue("id"), service.UpdateLabelInput{Name: req.Name, Color: req.Color})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toLabelResponse(label))
}

func (h *LabelHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
