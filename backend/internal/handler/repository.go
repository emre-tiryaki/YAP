package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type RepositoryHandler struct {
	session model.Session
	svc     *service.RepositoryService
}

func NewRepositoryHandler(session model.Session, svc *service.RepositoryService) *RepositoryHandler {
	return &RepositoryHandler{session: session, svc: svc}
}

func (h *RepositoryHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/resources/repositories", h.add)
	mux.HandleFunc("GET /api/resources/{id}/repository", h.get)
	mux.HandleFunc("POST /api/resources/{id}/repository/sync", h.sync)
	mux.HandleFunc("GET /api/resources/{id}/repository/entries", h.entries)
	mux.HandleFunc("GET /api/resources/{id}/repository/content", h.content)
	mux.HandleFunc("GET /api/resources/{id}/repository/commits", h.commits)
	mux.HandleFunc("GET /api/resources/{id}/repository/clones", h.clones)
	mux.HandleFunc("POST /api/resources/{id}/repository/clones", h.clone)
	mux.HandleFunc("DELETE /api/resources/{id}/repository/clones/{cloneId}", h.forgetClone)
	mux.HandleFunc("GET /api/local-folders", h.localFolders)
}

type repositoryResponse struct {
	ResourceID string  `json:"resource_id"`
	GitHubID   int64   `json:"github_id"`
	FullName   string  `json:"full_name"`
	HTMLURL    string  `json:"html_url"`
	Branch     string  `json:"branch"`
	Private    bool    `json:"private"`
	State      string  `json:"state"`
	CommitSHA  *string `json:"commit_sha"`
	FetchedAt  *string `json:"fetched_at"`
	CheckedAt  *string `json:"checked_at"`
	// AttemptedAt is the last fetch or update attempt, Error why it failed.
	AttemptedAt *string `json:"attempted_at"`
	Error       *string `json:"error"`
}

func toRepositoryResponse(r model.Repository) repositoryResponse {
	return repositoryResponse{
		ResourceID: r.ResourceID, GitHubID: r.GitHubID, FullName: r.FullName, HTMLURL: r.HTMLURL,
		Branch: r.Branch, Private: r.Private, State: string(r.State),
		CommitSHA: optionalString(r.CommitSHA), FetchedAt: formatTime(r.FetchedAt),
		CheckedAt: formatTime(r.CheckedAt), AttemptedAt: formatTime(r.AttemptedAt),
		Error: optionalString(string(r.Error)),
	}
}

type addRepositoryRequest struct {
	ParentID   *string `json:"parent_id"`
	FullName   string  `json:"full_name"`
	Name       string  `json:"name"`
	OnConflict string  `json:"on_conflict"`
}

func (h *RepositoryHandler) add(w http.ResponseWriter, r *http.Request) {
	var req addRepositoryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.svc.Add(r.Context(), h.session, service.AddRepositoryInput{
		ParentID:   deref(req.ParentID),
		FullName:   req.FullName,
		Name:       req.Name,
		OnConflict: onConflictParam(req.OnConflict),
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toResourceResponse(res))
}

func (h *RepositoryHandler) get(w http.ResponseWriter, r *http.Request) {
	repo, err := h.svc.Get(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toRepositoryResponse(repo))
}

func (h *RepositoryHandler) sync(w http.ResponseWriter, r *http.Request) {
	repo, err := h.svc.Sync(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, toRepositoryResponse(repo))
}

type repoEntryResponse struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	Size     int64  `json:"size"`
	MimeType string `json:"mime_type"`
	IsText   bool   `json:"is_text"`
	ModTime  string `json:"modified_at"`
}

func (h *RepositoryHandler) entries(w http.ResponseWriter, r *http.Request) {
	entries, err := h.svc.Entries(r.Context(), h.session, r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]repoEntryResponse, 0, len(entries))
	for _, e := range entries {
		res = append(res, repoEntryResponse{
			Name: e.Name, Path: e.Path, Kind: string(e.Kind), Size: e.Size,
			MimeType: e.MimeType, IsText: e.IsText, ModTime: rfc3339(e.ModTime),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

// content serves a file of the repository's copy, like a resource's content.
func (h *RepositoryHandler) content(w http.ResponseWriter, r *http.Request) {
	entry, f, err := h.svc.OpenFile(r.Context(), h.session, r.PathValue("id"), r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	defer f.Close()
	serveFile(w, r, servedFile{
		Name: entry.Name, MimeType: entry.MimeType, IsText: entry.IsText, ModTime: entry.ModTime,
		ETag: `"` + strconv.FormatInt(entry.ModTime.UnixNano(), 36) + "-" + strconv.FormatInt(entry.Size, 36) + `"`,
	}, f)
}

type cloneResponse struct {
	ID        string `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
	// Exists is false when the folder was deleted or moved since.
	Exists bool `json:"exists"`
}

func toCloneResponse(c model.RepositoryClone) cloneResponse {
	return cloneResponse{ID: c.ID, Path: c.Path, CreatedAt: rfc3339(c.CreatedAt), Exists: c.Exists}
}

func (h *RepositoryHandler) clones(w http.ResponseWriter, r *http.Request) {
	clones, err := h.svc.Clones(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]cloneResponse, 0, len(clones))
	for _, c := range clones {
		res = append(res, toCloneResponse(c))
	}
	writeJSON(w, http.StatusOK, res)
}

type cloneRequest struct {
	ParentPath string `json:"parent_path"`
	Name       string `json:"name"`
}

// clone writes to the user's disk, so it only takes JSON: a page on another
// site cannot send that to localhost without a CORS preflight, which this
// server does not answer.
func (h *RepositoryHandler) clone(w http.ResponseWriter, r *http.Request) {
	if mt := r.Header.Get("Content-Type"); !strings.HasPrefix(mt, "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "invalid_input"})
		return
	}
	var req cloneRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := h.svc.Clone(r.Context(), h.session, r.PathValue("id"), service.CloneInput{ParentPath: req.ParentPath, Name: req.Name})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toCloneResponse(c))
}

func (h *RepositoryHandler) forgetClone(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ForgetClone(r.Context(), h.session, r.PathValue("id"), r.PathValue("cloneId")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type localFolderResponse struct {
	Path    string   `json:"path"`
	Parent  *string  `json:"parent"`
	Home    string   `json:"home"`
	Folders []string `json:"folders"`
}

func (h *RepositoryHandler) localFolders(w http.ResponseWriter, r *http.Request) {
	f, err := h.svc.LocalFolders(r.Context(), r.URL.Query().Get("path"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, localFolderResponse{Path: f.Path, Parent: optionalString(f.Parent), Home: f.Home, Folders: f.Folders})
}

type commitResponse struct {
	SHA         string   `json:"sha"`
	Parents     []string `json:"parents"`
	Message     string   `json:"message"`
	AuthorName  string   `json:"author_name"`
	AuthorLogin string   `json:"author_login"`
	AvatarURL   string   `json:"avatar_url"`
	Date        string   `json:"date"`
	HTMLURL     string   `json:"html_url"`
}

func (h *RepositoryHandler) commits(w http.ResponseWriter, r *http.Request) {
	page := 1
	if v := r.URL.Query().Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_input"})
			return
		}
		page = n
	}
	commits, err := h.svc.Commits(r.Context(), h.session, r.PathValue("id"), page)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]commitResponse, 0, len(commits))
	for _, c := range commits {
		res = append(res, commitResponse{
			SHA: c.SHA, Parents: c.Parents, Message: c.Message, AuthorName: c.AuthorName,
			AuthorLogin: c.AuthorLogin, AvatarURL: c.AvatarURL, Date: rfc3339(c.Date), HTMLURL: c.HTMLURL,
		})
	}
	writeJSON(w, http.StatusOK, res)
}
