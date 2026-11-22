package service

import (
	"context"
	"slices"
	"testing"
)

func labelNames(t *testing.T, e accountEnv) map[string][]string {
	t.Helper()
	rows, err := e.db.Query(`SELECT t.title, l.name FROM task_labels tl
		JOIN tasks t ON t.id = tl.task_id JOIN labels l ON l.id = tl.label_id
		WHERE tl.removed_at IS NULL AND t.deleted_at IS NULL AND l.deleted_at IS NULL ORDER BY t.title, l.name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var task, label string
		if err := rows.Scan(&task, &label); err != nil {
			t.Fatal(err)
		}
		out[task] = append(out[task], label)
	}
	return out
}

func (e accountEnv) labelTask(t *testing.T, names ...string) {
	t.Helper()
	ctx := context.Background()
	var projectID, taskID string
	if err := e.db.QueryRow(`SELECT project_id, id FROM tasks WHERE deleted_at IS NULL`).Scan(&projectID, &taskID); err != nil {
		t.Fatal(err)
	}
	labels := NewLabelService(e.db, e.projectsRepo(), labelRepo())
	var ids []string
	for _, name := range names {
		l, err := labels.Create(ctx, e.session, projectID, CreateLabelInput{Name: name, Color: "teal"})
		if err != nil {
			t.Fatalf("Create label error = %v", err)
		}
		ids = append(ids, l.ID)
	}
	if _, err := e.tasks.Update(ctx, e.session, taskID, UpdateTaskInput{LabelIDs: &ids}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
}

func TestImportKeepsLabelsAfterReset(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	e.labelTask(t, "bug", "docs")
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	var live int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM labels WHERE deleted_at IS NULL`).Scan(&live); err != nil || live != 0 {
		t.Fatalf("live labels after reset = %d (err %v), want 0", live, err)
	}

	e.importBytes(t, archive)

	if got := labelNames(t, e)["T"]; !slices.Equal(got, []string{"bug", "docs"}) {
		t.Errorf("task labels after import = %v, want [bug docs]", got)
	}
}

func TestImportMergesLabelsByName(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	e.labelTask(t, "bug")
	archive := e.exportBytes(t)
	// A second import into the same, still full workspace adds nothing.
	res := e.importBytes(t, archive)
	if res.Added.Labels != 0 {
		t.Errorf("Added.Labels = %d, want 0 when everything is already there", res.Added.Labels)
	}
	var live int
	if err := e.db.QueryRow(`SELECT COUNT(*) FROM labels WHERE deleted_at IS NULL`).Scan(&live); err != nil || live != 1 {
		t.Errorf("live labels = %d (err %v), want 1", live, err)
	}
}
