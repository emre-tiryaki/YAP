package handler

import (
	"net/http"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type CommentHandler struct {
	session model.Session
	svc     *service.CommentService
}

func NewCommentHandler(session model.Session, svc *service.CommentService) *CommentHandler {
	return &CommentHandler{session: session, svc: svc}
}

func (h *CommentHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tasks/{id}/comments", h.list)
	mux.HandleFunc("POST /api/tasks/{id}/comments", h.create)
	mux.HandleFunc("PATCH /api/comments/{id}", h.update)
	mux.HandleFunc("DELETE /api/comments/{id}", h.delete)
}

type commentResponse struct {
	ID         string `json:"id"`
	TaskID     string `json:"task_id"`
	AuthorID   string `json:"author_id"`
	AuthorName string `json:"author_name"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
	// IsOwn tells the client whether the current member may edit or delete it.
	IsOwn bool `json:"is_own"`
}

type commentRequest struct {
	Body string `json:"body"`
}

func (h *CommentHandler) toResponse(c model.Comment) commentResponse {
	return commentResponse{
		ID:         c.ID,
		TaskID:     c.TaskID,
		AuthorID:   c.AuthorID,
		AuthorName: c.AuthorName,
		Body:       c.Body,
		CreatedAt:  c.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  c.UpdatedAt.UTC().Format(time.RFC3339),
		IsOwn:      c.AuthorID == h.session.MembershipID,
	}
}

func (h *CommentHandler) list(w http.ResponseWriter, r *http.Request) {
	comments, err := h.svc.List(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]commentResponse, 0, len(comments))
	for _, c := range comments {
		res = append(res, h.toResponse(c))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request) {
	var req commentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	comment, err := h.svc.Create(r.Context(), h.session, r.PathValue("id"), req.Body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.toResponse(comment))
}

func (h *CommentHandler) update(w http.ResponseWriter, r *http.Request) {
	var req commentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	comment, err := h.svc.Update(r.Context(), h.session, r.PathValue("id"), req.Body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toResponse(comment))
}

func (h *CommentHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
