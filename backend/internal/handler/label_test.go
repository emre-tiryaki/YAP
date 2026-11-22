package handler

import (
	"net/http"
	"testing"
)

func TestLabelsOverHTTP(t *testing.T) {
	env := newTestEnv(t)
	project := createProject(t, env, "Backend")

	rec := env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/labels", `{"name":"bug","color":"red"}`)
	wantStatus(t, rec, http.StatusCreated)
	bug := decode[labelResponse](t, rec)
	if bug.ID == "" || bug.Name != "bug" || bug.Color != "red" || bug.ProjectID != project.ID {
		t.Errorf("label = %+v", bug)
	}

	wantStatus(t, env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/labels", `{"name":"bug","color":"blue"}`), http.StatusConflict)
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/labels", `{"name":"x","color":"neon"}`), http.StatusBadRequest)
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects/missing/labels", `{"name":"x","color":"red"}`), http.StatusNotFound)

	rec = env.do(t, http.MethodPatch, "/api/labels/"+bug.ID, `{"name":"defect","color":"pink"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[labelResponse](t, rec); got.Name != "defect" || got.Color != "pink" {
		t.Errorf("updated label = %+v", got)
	}

	rec = env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/labels", "")
	wantStatus(t, rec, http.StatusOK)
	if list := decode[[]labelResponse](t, rec); len(list) != 1 || list[0].Name != "defect" {
		t.Errorf("labels = %+v", list)
	}

	// Labels on a task.
	rec = env.do(t, http.MethodPost, "/api/projects/"+project.ID+"/tasks", `{"title":"T"}`)
	wantStatus(t, rec, http.StatusCreated)
	task := decode[taskResponse](t, rec)
	if task.LabelIDs == nil || len(task.LabelIDs) != 0 {
		t.Errorf("new task label_ids = %v, want []", task.LabelIDs)
	}
	rec = env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"label_ids":["`+bug.ID+`"]}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[planTaskResponse](t, rec); len(got.LabelIDs) != 1 || got.LabelIDs[0] != bug.ID {
		t.Errorf("updated label_ids = %v", got.LabelIDs)
	}
	wantStatus(t, env.do(t, http.MethodPatch, "/api/tasks/"+task.ID, `{"label_ids":["missing"]}`), http.StatusBadRequest)

	rec = env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans/"+task.PlanID+"/tasks", "")
	wantStatus(t, rec, http.StatusOK)
	if list := decode[[]planTaskResponse](t, rec); len(list) != 1 || len(list[0].LabelIDs) != 1 {
		t.Errorf("plan tasks = %+v, want one task with a label", list)
	}

	wantStatus(t, env.do(t, http.MethodDelete, "/api/labels/"+bug.ID, ""), http.StatusNoContent)
	wantStatus(t, env.do(t, http.MethodDelete, "/api/labels/"+bug.ID, ""), http.StatusNotFound)
	rec = env.do(t, http.MethodGet, "/api/projects/"+project.ID+"/plans/"+task.PlanID+"/tasks", "")
	if list := decode[[]planTaskResponse](t, rec); len(list) != 1 || len(list[0].LabelIDs) != 0 {
		t.Errorf("plan tasks after label delete = %+v, want the label gone", list)
	}
}
