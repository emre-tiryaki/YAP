package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func (h *TaskHandler) registerActivity(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tasks/{id}/activity", h.activity)
	mux.HandleFunc("GET /api/tasks/{id}/relations", h.listRelations)
	mux.HandleFunc("POST /api/tasks/{id}/relations", h.addRelation)
	mux.HandleFunc("DELETE /api/tasks/{id}/relations/{relatedId}", h.removeRelation)
}

type taskEventResponse struct {
	ID        string            `json:"id"`
	ActorID   string            `json:"actor_id"`
	Action    string            `json:"action"`
	Details   map[string]string `json:"details"`
	CreatedAt string            `json:"created_at"`
}

func (h *TaskHandler) activity(w http.ResponseWriter, r *http.Request) {
	events, err := h.svc.Activity(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]taskEventResponse, 0, len(events))
	for _, e := range events {
		details := e.Details
		if details == nil {
			details = map[string]string{}
		}
		res = append(res, taskEventResponse{
			ID:        e.ID,
			ActorID:   e.ActorID,
			Action:    string(e.Action),
			Details:   details,
			CreatedAt: e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

type taskRelationResponse struct {
	// View is how this task reads the relation: blocking, blocked_by,
	// relates, duplicate_of or duplicated_by.
	View        string  `json:"view"`
	TaskID      string  `json:"task_id"`
	Title       string  `json:"title"`
	Number      int     `json:"number"`
	ProjectID   string  `json:"project_id"`
	PlanID      string  `json:"plan_id"`
	CompletedAt *string `json:"completed_at"`
}

func (h *TaskHandler) listRelations(w http.ResponseWriter, r *http.Request) {
	relations, err := h.svc.Relations(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]taskRelationResponse, 0, len(relations))
	for _, rel := range relations {
		res = append(res, taskRelationResponse{
			View:        string(rel.View),
			TaskID:      rel.TaskID,
			Title:       rel.Title,
			Number:      rel.Number,
			ProjectID:   rel.ProjectID,
			PlanID:      rel.PlanID,
			CompletedAt: formatTime(rel.CompletedAt),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

type addRelationRequest struct {
	RelatedTaskID string `json:"related_task_id"`
	// Kind is blocks (this task blocks the related one), relates or
	// duplicates (this task duplicates the related one).
	Kind string `json:"kind"`
}

func (h *TaskHandler) addRelation(w http.ResponseWriter, r *http.Request) {
	var req addRelationRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	err := h.svc.AddRelation(r.Context(), h.session, r.PathValue("id"), req.RelatedTaskID, model.RelationKind(req.Kind))
	if err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) removeRelation(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveRelation(r.Context(), h.session, r.PathValue("id"), r.PathValue("relatedId")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
