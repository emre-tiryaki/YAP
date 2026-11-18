package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type GitHubHandler struct {
	session model.Session
	svc     *service.GitHubService
}

func NewGitHubHandler(session model.Session, svc *service.GitHubService) *GitHubHandler {
	return &GitHubHandler{session: session, svc: svc}
}

func (h *GitHubHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/github", h.status)
	mux.HandleFunc("POST /api/github/connect", h.startConnect)
	mux.HandleFunc("POST /api/github/connect/poll", h.pollConnect)
	mux.HandleFunc("DELETE /api/github/account", h.disconnect)
	mux.HandleFunc("GET /api/github/repos", h.repos)
}

type githubAccountResponse struct {
	ID          int64  `json:"id"`
	Login       string `json:"login"`
	Name        string `json:"name"`
	AvatarURL   string `json:"avatar_url"`
	Scopes      string `json:"scopes"`
	ConnectedAt string `json:"connected_at"`
}

type githubStatusResponse struct {
	Configured   bool                   `json:"configured"`
	GitAvailable bool                   `json:"git_available"`
	Account      *githubAccountResponse `json:"account"`
}

func toGitHubAccount(a *service.GitHubAccount) *githubAccountResponse {
	if a == nil {
		return nil
	}
	return &githubAccountResponse{
		ID: a.User.ID, Login: a.User.Login, Name: a.User.Name, AvatarURL: a.User.AvatarURL,
		Scopes: a.Scopes, ConnectedAt: rfc3339(a.ConnectedAt),
	}
}

func (h *GitHubHandler) status(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Status(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, githubStatusResponse{
		Configured: st.Configured, GitAvailable: st.GitAvailable, Account: toGitHubAccount(st.Account),
	})
}

type deviceFlowResponse struct {
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresAt       string `json:"expires_at"`
	IntervalSeconds int    `json:"interval_seconds"`
}

func (h *GitHubHandler) startConnect(w http.ResponseWriter, r *http.Request) {
	flow, err := h.svc.StartConnect(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, deviceFlowResponse{
		UserCode: flow.UserCode, VerificationURI: flow.VerificationURI,
		ExpiresAt: rfc3339(flow.ExpiresAt), IntervalSeconds: int(flow.Interval.Seconds()),
	})
}

type connectResultResponse struct {
	State   string                 `json:"state"`
	Account *githubAccountResponse `json:"account"`
}

func (h *GitHubHandler) pollConnect(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.PollConnect(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, connectResultResponse{State: string(res.State), Account: toGitHubAccount(res.Account)})
}

func (h *GitHubHandler) disconnect(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Disconnect(r.Context(), h.session); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type githubRepoResponse struct {
	ID            int64   `json:"id"`
	FullName      string  `json:"full_name"`
	Description   string  `json:"description"`
	HTMLURL       string  `json:"html_url"`
	DefaultBranch string  `json:"default_branch"`
	Private       bool    `json:"private"`
	PushedAt      *string `json:"pushed_at"`
}

func (h *GitHubHandler) repos(w http.ResponseWriter, r *http.Request) {
	repos, err := h.svc.Repos(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]githubRepoResponse, 0, len(repos))
	for _, repo := range repos {
		res = append(res, githubRepoResponse{
			ID: repo.ID, FullName: repo.FullName, Description: repo.Description, HTMLURL: repo.HTMLURL,
			DefaultBranch: repo.DefaultBranch, Private: repo.Private, PushedAt: formatTime(repo.PushedAt),
		})
	}
	writeJSON(w, http.StatusOK, res)
}
