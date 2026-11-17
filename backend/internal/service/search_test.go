package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/repository"
)

func TestSearch(t *testing.T) {
	env := newResourceEnv(t)
	ctx := context.Background()
	projectSvc := NewProjectService(env.db, repository.NewProjectRepository(), repository.NewSettingsRepository())
	taskSvc := NewTaskService(env.db, repository.NewProjectRepository(), repository.NewTaskRepository())
	searchSvc := NewSearchService(env.db, repository.NewSearchRepository())
	ws := env.session.WorkspaceID

	project, err := projectSvc.Create(ctx, ws, CreateProjectInput{Name: "Ödev Takibi", Icon: "📚"})
	if err != nil {
		t.Fatalf("Create project error = %v", err)
	}
	plans, err := projectSvc.Plans(ctx, ws, project.ID)
	if err != nil || len(plans) == 0 {
		t.Fatalf("Plans() = %v, %v", plans, err)
	}
	mainPlan := plans[0]
	newTask := func(projectID, planID, title, description string) model.Task {
		t.Helper()
		task, err := taskSvc.Create(ctx, env.session, projectID, CreateTaskInput{Title: title, Description: description, PlanID: planID})
		if err != nil {
			t.Fatalf("Create task %q error = %v", title, err)
		}
		return task
	}
	titled := newTask(project.ID, mainPlan.ID, "Matematik ödevi", "")
	described := newTask(project.ID, mainPlan.ID, "Kitap oku", "Fizik ÖDEVİ için notlar")
	subtask, err := taskSvc.CreateSubtask(ctx, env.session, titled.ID, "Ödev sorularını yazdır")
	if err != nil {
		t.Fatalf("CreateSubtask() error = %v", err)
	}
	deleted := newTask(project.ID, mainPlan.ID, "Silinmiş ödev", "")
	if err := taskSvc.Delete(ctx, env.session, deleted.ID); err != nil {
		t.Fatalf("Delete task error = %v", err)
	}

	gone, err := projectSvc.Create(ctx, ws, CreateProjectInput{Name: "Eski proje"})
	if err != nil {
		t.Fatalf("Create project error = %v", err)
	}
	gonePlans, _ := projectSvc.Plans(ctx, ws, gone.ID)
	newTask(gone.ID, gonePlans[0].ID, "Eski ödev", "")
	if err := projectSvc.Delete(ctx, ws, gone.ID); err != nil {
		t.Fatalf("Delete project error = %v", err)
	}

	folder := env.folder(t, "", "Ödevler")
	file := env.upload(t, folder.ID, "odev-listesi.md", "x")
	trashed := env.upload(t, "", "ödev-eski.txt", "x")
	if err := env.svc.MoveToTrash(ctx, env.session, trashed.ID); err != nil {
		t.Fatalf("MoveToTrash() error = %v", err)
	}

	tests := []struct {
		name          string
		query         string
		wantProjects  []string
		wantPlans     []string
		wantTasks     []string
		wantResources []string
	}{
		{
			name:         "case-insensitive beyond ascii, skips deleted and trashed",
			query:        "  ödev ",
			wantProjects: []string{project.ID},
			// Titles starting with the query first, then other titles, then descriptions.
			wantTasks:     []string{subtask.ID, titled.ID, described.ID},
			wantResources: []string{folder.ID},
		},
		{name: "upper case query", query: "ÖDEV TAKİBİ", wantProjects: []string{project.ID}},
		{name: "plan name", query: "mai", wantPlans: []string{mainPlan.ID}},
		{name: "ascii file name", query: "LISTESI", wantResources: []string{file.ID}},
		{name: "like wildcards are literal", query: "%"},
		{name: "blank query", query: "   "},
		{name: "no match", query: "zzz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := searchSvc.Search(ctx, env.session, tt.query)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}
			check := func(kind string, got, want []string) {
				t.Helper()
				if strings.Join(got, ",") != strings.Join(want, ",") {
					t.Errorf("%s = %v, want %v", kind, got, want)
				}
			}
			check("projects", ids(res.Projects, func(p model.SearchProject) string { return p.ID }), tt.wantProjects)
			check("plans", ids(res.Plans, func(p model.SearchPlan) string { return p.ID }), tt.wantPlans)
			check("tasks", ids(res.Tasks, func(t model.SearchTask) string { return t.ID }), tt.wantTasks)
			check("resources", ids(res.Resources, func(r model.SearchResource) string { return r.ID }), tt.wantResources)
		})
	}

	t.Run("result details", func(t *testing.T) {
		res, err := searchSvc.Search(ctx, env.session, "yazdır")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(res.Tasks) != 1 {
			t.Fatalf("tasks = %+v, want one", res.Tasks)
		}
		got := res.Tasks[0]
		if got.ParentID != titled.ID || got.ProjectName != project.Name || got.PlanID != mainPlan.ID || got.PlanName != mainPlan.Name {
			t.Errorf("task = %+v", got)
		}
		res, err = searchSvc.Search(ctx, env.session, "listesi")
		if err != nil {
			t.Fatalf("Search() error = %v", err)
		}
		if len(res.Resources) != 1 || res.Resources[0].ParentName != "Ödevler" || res.Resources[0].Kind != model.ResourceFile {
			t.Errorf("resources = %+v", res.Resources)
		}
	})

	t.Run("too long query", func(t *testing.T) {
		_, err := searchSvc.Search(ctx, env.session, strings.Repeat("a", maxSearchQueryRunes+1))
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("Search() error = %v, want ErrInvalidInput", err)
		}
	})
}

func ids[T any](items []T, id func(T) string) []string {
	var out []string
	for _, item := range items {
		out = append(out, id(item))
	}
	return out
}
