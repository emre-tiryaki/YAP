package service

import (
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func taskPriorities(t *testing.T, e accountEnv) []string {
	t.Helper()
	rows, err := e.db.Query(`SELECT priority FROM tasks WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

func TestImportKeepsTaskPriority(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	high := model.PriorityHigh
	var id string
	if err := e.db.QueryRow(`SELECT id FROM tasks WHERE deleted_at IS NULL`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := e.tasks.Update(e.ctx, e.session, id, UpdateTaskInput{Priority: &high}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	e.importBytes(t, archive)

	if got := taskPriorities(t, e); len(got) != 1 || got[0] != "high" {
		t.Errorf("priorities after import = %v, want [high]", got)
	}
}

func TestImportOfOlderExportDefaultsPriority(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	archive := rewrite(t, e.exportBytes(t), func(doc map[string]any) {
		for _, row := range tableRows(doc, "tasks") {
			delete(row.(map[string]any), "priority")
		}
	})
	if err := e.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	e.importBytes(t, archive)

	if got := taskPriorities(t, e); len(got) != 1 || got[0] != "none" {
		t.Errorf("priorities after import = %v, want [none]", got)
	}
}
