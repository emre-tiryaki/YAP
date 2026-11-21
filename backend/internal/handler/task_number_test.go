package handler

import (
	"net/http"
	"testing"
)

func TestProjectIdentifierOverHTTP(t *testing.T) {
	env := newTestEnv(t)
	first := createProject(t, env, "Kitap")
	second := createProject(t, env, "Web")
	if first.Identifier != "KIT" || second.Identifier != "WEB" {
		t.Fatalf("identifiers = %q, %q; want KIT, WEB", first.Identifier, second.Identifier)
	}

	rec := env.do(t, http.MethodPatch, "/api/projects/"+second.ID, `{"identifier":"kit"}`)
	wantStatus(t, rec, http.StatusConflict)

	rec = env.do(t, http.MethodPatch, "/api/projects/"+second.ID, `{"identifier":"W"}`)
	wantStatus(t, rec, http.StatusBadRequest)

	rec = env.do(t, http.MethodPatch, "/api/projects/"+second.ID, `{"identifier":"site"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[projectResponse](t, rec); got.Identifier != "SITE" {
		t.Errorf("identifier = %q, want SITE", got.Identifier)
	}
}

func TestTaskNumbersOverHTTP(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Kitap")

	var created []taskResponse
	for _, title := range []string{"One", "Two"} {
		rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"`+title+`"}`)
		wantStatus(t, rec, http.StatusCreated)
		created = append(created, decode[taskResponse](t, rec))
	}
	if created[0].Number != 1 || created[1].Number != 2 {
		t.Errorf("numbers = %d, %d; want 1, 2", created[0].Number, created[1].Number)
	}

	rec := env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans/"+created[0].PlanID+"/tasks", "")
	wantStatus(t, rec, http.StatusOK)
	list := decode[[]planTaskResponse](t, rec)
	if len(list) != 2 || list[0].Number == 0 || list[1].Number == 0 {
		t.Errorf("plan tasks = %+v, want numbered tasks", list)
	}
}
