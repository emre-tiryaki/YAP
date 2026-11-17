package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type SearchHandler struct {
	session model.Session
	svc     *service.SearchService
}

func NewSearchHandler(session model.Session, svc *service.SearchService) *SearchHandler {
	return &SearchHandler{session: session, svc: svc}
}

func (h *SearchHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/search", h.search)
}

type searchProjectResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type searchPlanResponse struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	Name        string `json:"name"`
	Icon        string `json:"icon"`
}

type searchTaskResponse struct {
	ID          string  `json:"id"`
	ProjectID   string  `json:"project_id"`
	ProjectName string  `json:"project_name"`
	PlanID      string  `json:"plan_id"`
	PlanName    string  `json:"plan_name"`
	ParentID    *string `json:"parent_id"`
	Title       string  `json:"title"`
	Done        bool    `json:"done"`
}

type searchResourceResponse struct {
	ID         string  `json:"id"`
	ParentID   *string `json:"parent_id"`
	ParentName *string `json:"parent_name"`
	Kind       string  `json:"kind"`
	Name       string  `json:"name"`
	MimeType   string  `json:"mime_type"`
	IsText     bool    `json:"is_text"`
}

type searchResponse struct {
	Projects  []searchProjectResponse  `json:"projects"`
	Plans     []searchPlanResponse     `json:"plans"`
	Tasks     []searchTaskResponse     `json:"tasks"`
	Resources []searchResourceResponse `json:"resources"`
}

func toSearchResponse(res model.SearchResults) searchResponse {
	out := searchResponse{
		Projects:  make([]searchProjectResponse, 0, len(res.Projects)),
		Plans:     make([]searchPlanResponse, 0, len(res.Plans)),
		Tasks:     make([]searchTaskResponse, 0, len(res.Tasks)),
		Resources: make([]searchResourceResponse, 0, len(res.Resources)),
	}
	for _, p := range res.Projects {
		out.Projects = append(out.Projects, searchProjectResponse{ID: p.ID, Name: p.Name, Icon: p.Icon})
	}
	for _, p := range res.Plans {
		out.Plans = append(out.Plans, searchPlanResponse{
			ID: p.ID, ProjectID: p.ProjectID, ProjectName: p.ProjectName, Name: p.Name, Icon: p.Icon,
		})
	}
	for _, t := range res.Tasks {
		out.Tasks = append(out.Tasks, searchTaskResponse{
			ID: t.ID, ProjectID: t.ProjectID, ProjectName: t.ProjectName, PlanID: t.PlanID,
			PlanName: t.PlanName, ParentID: optionalString(t.ParentID), Title: t.Title, Done: t.Done,
		})
	}
	for _, r := range res.Resources {
		out.Resources = append(out.Resources, searchResourceResponse{
			ID: r.ID, ParentID: optionalString(r.ParentID), ParentName: optionalString(r.ParentName),
			Kind: string(r.Kind), Name: r.Name, MimeType: r.MimeType, IsText: r.IsText,
		})
	}
	return out
}

func (h *SearchHandler) search(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.Search(r.Context(), h.session, r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSearchResponse(res))
}
