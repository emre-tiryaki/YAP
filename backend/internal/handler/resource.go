package handler

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type ResourceHandler struct {
	session model.Session
	svc     *service.ResourceService
}

func NewResourceHandler(session model.Session, svc *service.ResourceService) *ResourceHandler {
	return &ResourceHandler{session: session, svc: svc}
}

func (h *ResourceHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/resources", h.list)
	mux.HandleFunc("GET /api/resources/trash", h.trash)
	mux.HandleFunc("DELETE /api/resources/trash", h.emptyTrash)
	mux.HandleFunc("GET /api/resources/events", h.workspaceEvents)
	mux.HandleFunc("POST /api/resources/folders", h.createFolder)
	mux.HandleFunc("POST /api/resources/files", h.createFile)
	mux.HandleFunc("POST /api/resources/upload", h.upload)
	mux.HandleFunc("GET /api/resources/{id}", h.get)
	mux.HandleFunc("PATCH /api/resources/{id}", h.rename)
	mux.HandleFunc("DELETE /api/resources/{id}", h.purge)
	mux.HandleFunc("GET /api/resources/{id}/content", h.content)
	mux.HandleFunc("PUT /api/resources/{id}/content", h.saveContent)
	mux.HandleFunc("POST /api/resources/{id}/move", h.move)
	mux.HandleFunc("POST /api/resources/{id}/copy", h.copy)
	mux.HandleFunc("POST /api/resources/{id}/trash", h.moveToTrash)
	mux.HandleFunc("POST /api/resources/{id}/restore", h.restore)
	mux.HandleFunc("GET /api/resources/{id}/events", h.resourceEvents)
	mux.HandleFunc("GET /api/resources/{id}/tasks", h.resourceTasks)
	mux.HandleFunc("GET /api/tasks/{id}/resources", h.taskResources)
	mux.HandleFunc("PUT /api/tasks/{id}/resources/{resourceId}", h.link)
	mux.HandleFunc("DELETE /api/tasks/{id}/resources/{resourceId}", h.unlink)
	mux.HandleFunc("GET /api/projects/{id}/resource-events", h.projectEvents)
}

type resourceResponse struct {
	ID        string  `json:"id"`
	ParentID  *string `json:"parent_id"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	MimeType  string  `json:"mime_type"`
	Size      int64   `json:"size"`
	IsText    bool    `json:"is_text"`
	Revision  int64   `json:"revision"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
	TrashedAt *string `json:"trashed_at"`
	// TrashedRootID is the item that was moved to the trash, taking this one
	// with it. Only an item that is its own trash root can be restored.
	TrashedRootID *string `json:"trashed_root_id"`
	// IsRepository marks a read-only folder filled with a GitHub repository.
	IsRepository bool `json:"is_repository"`
}

func rfc3339(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func toResourceResponse(r model.Resource) resourceResponse {
	res := resourceResponse{
		ID:        r.ID,
		ParentID:  optionalString(r.ParentID),
		Kind:      string(r.Kind),
		Name:      r.Name,
		MimeType:  r.MimeType,
		Size:      r.Size,
		IsText:    r.IsText,
		Revision:  r.Revision,
		CreatedAt: rfc3339(r.CreatedAt),
		UpdatedAt: rfc3339(r.UpdatedAt),
	}
	res.TrashedAt = formatTime(r.TrashedAt)
	res.TrashedRootID = optionalString(r.TrashedRootID)
	res.IsRepository = r.IsRepository
	return res
}

func writeResources(w http.ResponseWriter, resources []model.Resource) {
	res := make([]resourceResponse, 0, len(resources))
	for _, r := range resources {
		res = append(res, toResourceResponse(r))
	}
	writeJSON(w, http.StatusOK, res)
}

type resourceEventResponse struct {
	ID           string          `json:"id"`
	ResourceID   string          `json:"resource_id"`
	ResourceKind string          `json:"resource_kind"`
	ActorID      string          `json:"actor_id"`
	ActorName    string          `json:"actor_name"`
	Action       string          `json:"action"`
	TaskID       *string         `json:"task_id"`
	Details      json.RawMessage `json:"details"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

type linkedTaskResponse struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	PlanID      string  `json:"plan_id"`
	CompletedAt *string `json:"completed_at"`
}

func (h *ResourceHandler) list(w http.ResponseWriter, r *http.Request) {
	resources, err := h.svc.List(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeResources(w, resources)
}

func (h *ResourceHandler) trash(w http.ResponseWriter, r *http.Request) {
	resources, err := h.svc.Trash(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeResources(w, resources)
}

func (h *ResourceHandler) emptyTrash(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.EmptyTrash(r.Context(), h.session); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ResourceHandler) get(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Get(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResourceResponse(res))
}

type createResourceRequest struct {
	ParentID *string `json:"parent_id"`
	Name     string  `json:"name"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (h *ResourceHandler) createFolder(w http.ResponseWriter, r *http.Request) {
	var req createResourceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.CreateFolder(r.Context(), h.session, deref(req.ParentID), req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResourceResponse(res))
}

func (h *ResourceHandler) createFile(w http.ResponseWriter, r *http.Request) {
	var req createResourceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.CreateFile(r.Context(), h.session, deref(req.ParentID), req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResourceResponse(res))
}

func onConflictParam(v string) service.OnConflict {
	if v == "" {
		return service.ConflictFail
	}
	return service.OnConflict(v)
}

// upload takes the raw file bytes as the body; the target folder, name and
// conflict policy come in the query string.
func (h *ResourceHandler) upload(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	body := http.MaxBytesReader(w, r.Body, service.MaxUploadBytes+1)
	res, err := h.svc.Upload(r.Context(), h.session, service.UploadInput{
		ParentID:    q.Get("parent_id"),
		Name:        q.Get("name"),
		OnConflict:  onConflictParam(q.Get("on_conflict")),
		ContentType: r.Header.Get("Content-Type"),
	}, body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResourceResponse(res))
}

// content serves the bytes of a file (see serveFile).
func (h *ResourceHandler) content(w http.ResponseWriter, r *http.Request) {
	res, f, err := h.svc.Open(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer f.Close()
	serveFile(w, r, servedFile{
		Name: res.Name, MimeType: res.MimeType, IsText: res.IsText, ModTime: res.UpdatedAt,
		ETag: `"` + res.ID + "-" + strconv.FormatInt(res.Revision, 10) + `"`,
	}, f)
}

type servedFile struct {
	Name     string
	MimeType string
	IsText   bool
	ModTime  time.Time
	ETag     string
}

// serveFile sends file bytes for preview or download. Range requests work,
// so audio and video can seek. The response is sandboxed so an HTML or SVG
// file cannot run script with the app's origin; PDFs are left out because
// the browser's viewer does not load in a sandbox.
func serveFile(w http.ResponseWriter, r *http.Request, file servedFile, content io.ReadSeeker) {
	contentType := file.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if file.IsText && (strings.HasPrefix(contentType, "text/") || contentType == "application/octet-stream") {
		if contentType == "application/octet-stream" {
			contentType = "text/plain"
		}
		contentType += "; charset=utf-8"
	}
	header := w.Header()
	header.Set("Content-Type", contentType)
	header.Set("X-Content-Type-Options", "nosniff")
	if file.MimeType != "application/pdf" {
		header.Set("Content-Security-Policy", "sandbox; default-src 'none'; img-src 'self' data:; media-src 'self'; style-src 'unsafe-inline'")
	}
	header.Set("Cache-Control", "no-cache")
	if file.ETag != "" {
		header.Set("ETag", file.ETag)
	}
	disposition := "inline"
	if r.URL.Query().Get("download") == "1" {
		disposition = "attachment"
	}
	header.Set("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.Name}))
	http.ServeContent(w, r, "", file.ModTime, content)
}

// saveContent replaces the text of a file. The revision the edit is based on
// comes in the query string; a mismatch answers 409 "stale".
func (h *ResourceHandler) saveContent(w http.ResponseWriter, r *http.Request) {
	revision, err := strconv.ParseInt(r.URL.Query().Get("revision"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_input"})
		return
	}
	body := http.MaxBytesReader(w, r.Body, service.MaxTextBytes+1)
	res, err := h.svc.SaveContent(r.Context(), h.session, r.PathValue("id"), revision, body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResourceResponse(res))
}

type renameResourceRequest struct {
	Name string `json:"name"`
}

func (h *ResourceHandler) rename(w http.ResponseWriter, r *http.Request) {
	var req renameResourceRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.Rename(r.Context(), h.session, r.PathValue("id"), req.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResourceResponse(res))
}

type transferRequest struct {
	ParentID   *string `json:"parent_id"`
	OnConflict string  `json:"on_conflict"`
}

func (h *ResourceHandler) transfer(w http.ResponseWriter, r *http.Request,
	do func(*http.Request, string, service.TransferInput) (model.Resource, error), status int) {
	var req transferRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := do(r, r.PathValue("id"), service.TransferInput{
		ParentID:   deref(req.ParentID),
		OnConflict: onConflictParam(req.OnConflict),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, status, toResourceResponse(res))
}

func (h *ResourceHandler) move(w http.ResponseWriter, r *http.Request) {
	h.transfer(w, r, func(r *http.Request, id string, in service.TransferInput) (model.Resource, error) {
		return h.svc.Move(r.Context(), h.session, id, in)
	}, http.StatusOK)
}

func (h *ResourceHandler) copy(w http.ResponseWriter, r *http.Request) {
	h.transfer(w, r, func(r *http.Request, id string, in service.TransferInput) (model.Resource, error) {
		return h.svc.Copy(r.Context(), h.session, id, in)
	}, http.StatusCreated)
}

func (h *ResourceHandler) moveToTrash(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MoveToTrash(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ResourceHandler) restore(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Restore(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toResourceResponse(res))
}

func (h *ResourceHandler) purge(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Purge(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ResourceHandler) events(w http.ResponseWriter, r *http.Request, in service.EventQuery) {
	q := r.URL.Query()
	in.Before = q.Get("before")
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_input"})
			return
		}
		in.Limit = n
	}
	events, err := h.svc.Events(r.Context(), h.session, in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]resourceEventResponse, 0, len(events))
	for _, e := range events {
		res = append(res, resourceEventResponse{
			ID:           e.ID,
			ResourceID:   e.ResourceID,
			ResourceKind: string(e.ResourceKind),
			ActorID:      e.ActorID,
			ActorName:    e.ActorName,
			Action:       string(e.Action),
			TaskID:       optionalString(e.TaskID),
			Details:      json.RawMessage(e.Details),
			CreatedAt:    rfc3339(e.CreatedAt),
			UpdatedAt:    rfc3339(e.UpdatedAt),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ResourceHandler) workspaceEvents(w http.ResponseWriter, r *http.Request) {
	h.events(w, r, service.EventQuery{})
}

func (h *ResourceHandler) resourceEvents(w http.ResponseWriter, r *http.Request) {
	h.events(w, r, service.EventQuery{ResourceID: r.PathValue("id")})
}

func (h *ResourceHandler) projectEvents(w http.ResponseWriter, r *http.Request) {
	h.events(w, r, service.EventQuery{ProjectID: r.PathValue("id")})
}

func (h *ResourceHandler) resourceTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ResourceTasks(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]linkedTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		res = append(res, linkedTaskResponse{
			ID: t.TaskID, Title: t.Title, ProjectID: t.ProjectID, ProjectName: t.ProjectName,
			PlanID: t.PlanID, CompletedAt: formatTime(t.CompletedAt),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ResourceHandler) taskResources(w http.ResponseWriter, r *http.Request) {
	resources, err := h.svc.TaskResources(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeResources(w, resources)
}

func (h *ResourceHandler) link(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Link(r.Context(), h.session, r.PathValue("id"), r.PathValue("resourceId")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ResourceHandler) unlink(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Unlink(r.Context(), h.session, r.PathValue("id"), r.PathValue("resourceId")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
