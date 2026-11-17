package handler

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSearch(t *testing.T) {
	env := newTestEnv(t)
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects", `{"name":"Ödev Takibi","icon":"📚"}`), http.StatusCreated)

	tests := []struct {
		name         string
		query        string
		wantStatus   int
		wantProjects int
	}{
		{name: "match", query: "ödev", wantStatus: http.StatusOK, wantProjects: 1},
		{name: "no match", query: "zzz", wantStatus: http.StatusOK},
		{name: "empty", query: "", wantStatus: http.StatusOK},
		{name: "too long", query: strings.Repeat("a", 201), wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := env.do(t, http.MethodGet, "/api/search?q="+url.QueryEscape(tt.query), "")
			wantStatus(t, rec, tt.wantStatus)
			if tt.wantStatus != http.StatusOK {
				return
			}
			got := decode[searchResponse](t, rec)
			if got.Projects == nil || got.Plans == nil || got.Tasks == nil || got.Resources == nil {
				t.Fatalf("groups must be arrays, got %s", rec.Body.String())
			}
			if len(got.Projects) != tt.wantProjects {
				t.Fatalf("projects = %+v, want %d", got.Projects, tt.wantProjects)
			}
			if tt.wantProjects > 0 && (got.Projects[0].Name != "Ödev Takibi" || got.Projects[0].Icon != "📚") {
				t.Errorf("project = %+v", got.Projects[0])
			}
		})
	}
}
