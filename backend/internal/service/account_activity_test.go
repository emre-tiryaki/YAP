package service

import (
	"errors"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

type activitySnapshot struct {
	events    map[string][]string // task title -> actions in order
	relations []string            // "A blocks B"
	start     map[string]string   // task title -> start date
}

func takeActivity(t *testing.T, e accountEnv) activitySnapshot {
	t.Helper()
	snap := activitySnapshot{events: map[string][]string{}, start: map[string]string{}}
	rows, err := e.db.Query(`SELECT t.title, ev.action FROM task_events ev JOIN tasks t ON t.id = ev.task_id
		WHERE ev.deleted_at IS NULL AND t.deleted_at IS NULL ORDER BY t.title, ev.created_at, ev.id`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var title, action string
		if err := rows.Scan(&title, &action); err != nil {
			t.Fatal(err)
		}
		snap.events[title] = append(snap.events[title], action)
	}
	rows.Close()
	rows, err = e.db.Query(`SELECT a.title || ' ' || r.kind || ' ' || b.title FROM task_relations r
		JOIN tasks a ON a.id = r.task_id JOIN tasks b ON b.id = r.related_task_id
		WHERE r.removed_at IS NULL AND a.deleted_at IS NULL AND b.deleted_at IS NULL ORDER BY 1`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var rel string
		if err := rows.Scan(&rel); err != nil {
			t.Fatal(err)
		}
		snap.relations = append(snap.relations, rel)
	}
	rows.Close()
	rows, err = e.db.Query(`SELECT title, start_date FROM tasks WHERE deleted_at IS NULL AND start_date IS NOT NULL`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var title, start string
		if err := rows.Scan(&title, &start); err != nil {
			t.Fatal(err)
		}
		snap.start[title] = start
	}
	rows.Close()
	return snap
}

func TestImportKeepsActivityRelationsAndStartDate(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t) // creates task "T" in project "Kept"
	var projectID string
	if err := e.db.QueryRow(`SELECT project_id FROM tasks WHERE deleted_at IS NULL`).Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	var taskT model.Task
	if err := e.db.QueryRow(`SELECT id FROM tasks WHERE title = 'T'`).Scan(&taskT.ID); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	due := time.Date(2026, 12, 9, 0, 0, 0, 0, time.UTC)
	other, err := e.tasks.Create(e.ctx, e.session, projectID, CreateTaskInput{Title: "U", StartDate: &start, DueDate: &due})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	high := model.PriorityHigh
	if _, err := e.tasks.Update(e.ctx, e.session, taskT.ID, UpdateTaskInput{Priority: &high}); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if err := e.tasks.AddRelation(e.ctx, e.session, taskT.ID, other.ID, model.RelationBlocks); err != nil {
		t.Fatalf("AddRelation() error = %v", err)
	}

	before := takeActivity(t, e)
	if len(before.events["T"]) == 0 || len(before.relations) != 1 || before.start["U"] == "" {
		t.Fatalf("seeded snapshot is empty: %+v", before)
	}
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session, ResetOptions{}); err != nil {
		t.Fatal(err)
	}
	if got := takeActivity(t, e); len(got.events) != 0 || len(got.relations) != 0 {
		t.Fatalf("after reset: %+v, want no events or relations", got)
	}

	e.importBytes(t, archive)

	after := takeActivity(t, e)
	for title, want := range before.events {
		if !equalStrings(after.events[title], want) {
			t.Errorf("events of %s = %v, want %v", title, after.events[title], want)
		}
	}
	if !equalStrings(after.relations, before.relations) {
		t.Errorf("relations = %v, want %v", after.relations, before.relations)
	}
	if after.start["U"] != before.start["U"] {
		t.Errorf("start date = %q, want %q", after.start["U"], before.start["U"])
	}

	// Importing the same archive again adds nothing.
	e.importBytes(t, archive)
	again := takeActivity(t, e)
	if !equalStrings(again.relations, before.relations) || len(again.events["T"]) != len(before.events["T"]) {
		t.Errorf("after a second import: %+v, want the same as before", again)
	}
}

func TestAddRelationRules(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	var projectID, taskID string
	if err := e.db.QueryRow(`SELECT project_id, id FROM tasks WHERE deleted_at IS NULL`).Scan(&projectID, &taskID); err != nil {
		t.Fatal(err)
	}
	other, err := e.tasks.Create(e.ctx, e.session, projectID, CreateTaskInput{Title: "U"})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		from    string
		to      string
		kind    model.RelationKind
		wantErr error
	}{
		{"valid", taskID, other.ID, model.RelationDuplicates, nil},
		{"duplicate pair", taskID, other.ID, model.RelationRelates, ErrRelationExists},
		{"reverse pair", other.ID, taskID, model.RelationBlocks, ErrRelationExists},
		{"self", taskID, taskID, model.RelationBlocks, ErrInvalidInput},
		{"bad kind", taskID, other.ID, "x", ErrInvalidInput},
		{"missing", taskID, "nope", model.RelationBlocks, ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := e.tasks.AddRelation(e.ctx, e.session, tt.from, tt.to, tt.kind)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("AddRelation() error = %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("AddRelation() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
