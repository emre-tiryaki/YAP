package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func TestTaskFirstStatusID(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewTaskRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)

	got, err := repo.FirstStatusID(ctx, db, s.WorkspaceID, p.ID, model.StatusInProgress)
	if err != nil {
		t.Fatalf("FirstStatusID() error = %v", err)
	}
	if got != statuses[model.StatusInProgress] {
		t.Errorf("FirstStatusID() = %q, want %q", got, statuses[model.StatusInProgress])
	}

	extra := model.Status{
		ID: model.NewID(), WorkspaceID: s.WorkspaceID, ProjectID: p.ID,
		Title: "Earlier", Color: "#000", Category: model.StatusToDo, Position: "a0",
	}
	if err := NewProjectRepository().InsertStatus(ctx, db, extra, baseNow); err != nil {
		t.Fatal(err)
	}
	got, err = repo.FirstStatusID(ctx, db, s.WorkspaceID, p.ID, model.StatusToDo)
	if err != nil {
		t.Fatal(err)
	}
	if got != extra.ID {
		t.Errorf("FirstStatusID() = %q, want lowest position %q", got, extra.ID)
	}

	mustExec(t, db, `UPDATE statuses SET deleted_at = ? WHERE id = ?`, baseNow, extra.ID)
	got, err = repo.FirstStatusID(ctx, db, s.WorkspaceID, p.ID, model.StatusToDo)
	if err != nil {
		t.Fatal(err)
	}
	if got != statuses[model.StatusToDo] {
		t.Errorf("FirstStatusID() = %q, want deleted status to be skipped", got)
	}
}

func TestTaskFirstStatusIDNotFound(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewTaskRepository()
	s := seedWorkspace(t, db, "owner")
	other := seedWorkspace(t, db, "other")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	seedStatuses(t, db, s.WorkspaceID, p.ID)

	tests := []struct {
		name        string
		workspaceID string
		projectID   string
	}{
		{"unknown project", s.WorkspaceID, "missing"},
		{"other workspace", other.WorkspaceID, p.ID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.FirstStatusID(ctx, db, tt.workspaceID, tt.projectID, model.StatusToDo)
			if !errors.Is(err, sql.ErrNoRows) {
				t.Errorf("FirstStatusID() error = %v, want sql.ErrNoRows", err)
			}
		})
	}
}

func TestTaskLastPosition(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewTaskRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)
	todo := statuses[model.StatusToDo]

	got, err := repo.LastPosition(ctx, db, s.WorkspaceID, p.ID, todo)
	if err != nil {
		t.Fatalf("LastPosition() error = %v", err)
	}
	if got != "" {
		t.Errorf("LastPosition() on empty column = %q, want empty", got)
	}

	seedTask(t, db, s, p.ID, todo, "a", "a1", nil)
	seedTask(t, db, s, p.ID, todo, "b", "a5", nil)
	seedTask(t, db, s, p.ID, statuses[model.StatusDone], "other column", "zz", nil)

	got, err = repo.LastPosition(ctx, db, s.WorkspaceID, p.ID, todo)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a5" {
		t.Errorf("LastPosition() = %q, want a5", got)
	}
}

func TestTaskInsert(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewTaskRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)

	due := time.Date(2026, 12, 1, 15, 4, 5, 0, time.FixedZone("TRT", 3*3600))
	tests := []struct {
		name        string
		description string
		due         *time.Time
		wantDesc    sql.NullString
		wantDue     sql.NullString
	}{
		{"full", "details", &due, sql.NullString{String: "details", Valid: true}, sql.NullString{String: "2026-12-01T12:04:05Z", Valid: true}},
		{"minimal", "", nil, sql.NullString{}, sql.NullString{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := model.Task{
				ID: model.NewID(), WorkspaceID: s.WorkspaceID, ProjectID: p.ID,
				StatusID: statuses[model.StatusToDo], CreatedBy: s.MembershipID,
				Title: tt.name, Description: tt.description, Position: "a0", DueDate: tt.due,
			}
			if err := repo.Insert(ctx, db, task, baseNow); err != nil {
				t.Fatalf("Insert() error = %v", err)
			}

			var title string
			var desc, dueDate sql.NullString
			var createdAt, updatedAt string
			err := db.QueryRowContext(ctx,
				`SELECT title, description, due_date, created_at, updated_at FROM tasks WHERE id = ?`, task.ID,
			).Scan(&title, &desc, &dueDate, &createdAt, &updatedAt)
			if err != nil {
				t.Fatal(err)
			}
			if title != tt.name || desc != tt.wantDesc || dueDate != tt.wantDue {
				t.Errorf("stored = %q %v %v", title, desc, dueDate)
			}
			if createdAt != baseNow || updatedAt != baseNow {
				t.Errorf("timestamps = %q %q, want %q", createdAt, updatedAt, baseNow)
			}
		})
	}
}

func TestTaskInsertErrors(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewTaskRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)

	valid := model.Task{
		ID: model.NewID(), WorkspaceID: s.WorkspaceID, ProjectID: p.ID,
		StatusID: statuses[model.StatusToDo], CreatedBy: s.MembershipID, Title: "t", Position: "a0",
	}
	if err := repo.Insert(ctx, db, valid, baseNow); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		mutate func(task *model.Task)
	}{
		{"duplicate id", func(task *model.Task) { task.ID = valid.ID }},
		{"unknown status", func(task *model.Task) { task.ID = model.NewID(); task.StatusID = model.NewID() }},
		{"unknown project", func(task *model.Task) { task.ID = model.NewID(); task.ProjectID = model.NewID() }},
		{"unknown creator", func(task *model.Task) { task.ID = model.NewID(); task.CreatedBy = model.NewID() }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bad := valid
			tt.mutate(&bad)
			if err := repo.Insert(ctx, db, bad, baseNow); err == nil {
				t.Error("Insert() expected an error")
			}
		})
	}
}

func TestTaskInsertAssignee(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewTaskRepository()
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)
	task := seedTask(t, db, s, p.ID, statuses[model.StatusToDo], "t", "a0", nil)

	if err := repo.InsertAssignee(ctx, db, s.WorkspaceID, task.ID, s.MembershipID, baseNow); err != nil {
		t.Fatalf("InsertAssignee() error = %v", err)
	}
	if err := repo.InsertAssignee(ctx, db, s.WorkspaceID, task.ID, s.MembershipID, baseNow); err == nil {
		t.Error("InsertAssignee() expected an error for a duplicate assignment")
	}
	if err := repo.InsertAssignee(ctx, db, s.WorkspaceID, task.ID, model.NewID(), baseNow); err == nil {
		t.Error("InsertAssignee() expected an error for an unknown membership")
	}

	var assignedAt string
	var removedAt sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT assigned_at, removed_at FROM task_assignees WHERE task_id = ? AND membership_id = ?`,
		task.ID, s.MembershipID,
	).Scan(&assignedAt, &removedAt)
	if err != nil {
		t.Fatal(err)
	}
	if assignedAt != baseNow || removedAt.Valid {
		t.Errorf("assignee row = %q %v", assignedAt, removedAt)
	}
}

func TestTaskListUpcoming(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	repo := NewTaskRepository()
	s := seedWorkspace(t, db, "owner")
	colleague := addMember(t, db, s.WorkspaceID)
	foreign := seedWorkspace(t, db, "foreign")

	p := seedProject(t, db, s.WorkspaceID, "Alpha", baseNow)
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)
	todo, doing, done := statuses[model.StatusToDo], statuses[model.StatusInProgress], statuses[model.StatusDone]

	d1 := time.Date(2026, 12, 1, 9, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 12, 2, 9, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 12, 3, 9, 0, 0, 0, time.UTC)

	second := seedTask(t, db, s, p.ID, doing, "second", "a1", &d2)
	first := seedTask(t, db, s, p.ID, todo, "first", "a2", &d1)
	third := seedTask(t, db, s, p.ID, todo, "third", "a3", &d3)
	finished := seedTask(t, db, s, p.ID, done, "finished", "a4", &d1)
	undated := seedTask(t, db, s, p.ID, todo, "undated", "a5", nil)
	deletedTask := seedTask(t, db, s, p.ID, todo, "deleted task", "a6", &d1)
	removedAssignee := seedTask(t, db, s, p.ID, todo, "removed assignee", "a7", &d1)
	notMine := seedTask(t, db, s, p.ID, todo, "not mine", "a8", &d1)

	deletedProject := seedProject(t, db, s.WorkspaceID, "Gone", baseNow)
	dpStatuses := seedStatuses(t, db, s.WorkspaceID, deletedProject.ID)
	inDeletedProject := seedTask(t, db, s, deletedProject.ID, dpStatuses[model.StatusToDo], "in deleted project", "a0", &d1)

	for _, task := range []model.Task{second, first, third, finished, undated, deletedTask, removedAssignee, inDeletedProject} {
		assign(t, db, s.WorkspaceID, task.ID, s.MembershipID)
	}
	assign(t, db, s.WorkspaceID, notMine.ID, colleague)

	fp := seedProject(t, db, foreign.WorkspaceID, "Foreign", baseNow)
	fs := seedStatuses(t, db, foreign.WorkspaceID, fp.ID)
	foreignTask := seedTask(t, db, foreign, fp.ID, fs[model.StatusToDo], "foreign task", "a0", &d1)
	assign(t, db, foreign.WorkspaceID, foreignTask.ID, foreign.MembershipID)

	mustExec(t, db, `UPDATE tasks SET deleted_at = ? WHERE id = ?`, baseNow, deletedTask.ID)
	mustExec(t, db, `UPDATE task_assignees SET removed_at = ? WHERE task_id = ?`, baseNow, removedAssignee.ID)
	mustExec(t, db, `UPDATE projects SET deleted_at = ? WHERE id = ?`, baseNow, deletedProject.ID)

	got, err := repo.ListUpcoming(ctx, db, s.WorkspaceID, s.MembershipID, 10)
	if err != nil {
		t.Fatalf("ListUpcoming() error = %v", err)
	}

	wantTitles := []string{"first", "second", "third"}
	if len(got) != len(wantTitles) {
		t.Fatalf("ListUpcoming() returned %d tasks (%+v), want %d", len(got), got, len(wantTitles))
	}
	for i, title := range wantTitles {
		if got[i].Title != title {
			t.Errorf("got[%d].Title = %q, want %q", i, got[i].Title, title)
		}
	}

	if got[0].ID != first.ID || got[0].ProjectID != p.ID || got[0].ProjectName != "Alpha" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[0].WorkspaceID != s.WorkspaceID {
		t.Errorf("WorkspaceID = %q, want %q", got[0].WorkspaceID, s.WorkspaceID)
	}
	if got[0].StatusCategory != model.StatusToDo || got[1].StatusCategory != model.StatusInProgress {
		t.Errorf("categories = %q %q", got[0].StatusCategory, got[1].StatusCategory)
	}
	if got[0].DueDate == nil || !got[0].DueDate.Equal(d1) {
		t.Errorf("DueDate = %v, want %v", got[0].DueDate, d1)
	}

	limited, err := repo.ListUpcoming(ctx, db, s.WorkspaceID, s.MembershipID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 2 || limited[0].Title != "first" || limited[1].Title != "second" {
		t.Errorf("limited = %+v", limited)
	}

	colleagueTasks, err := repo.ListUpcoming(ctx, db, s.WorkspaceID, colleague, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(colleagueTasks) != 1 || colleagueTasks[0].Title != "not mine" {
		t.Errorf("colleague tasks = %+v", colleagueTasks)
	}
}

func TestTaskListUpcomingEmptyIsNonNil(t *testing.T) {
	db := newTestDB(t)
	s := seedWorkspace(t, db, "owner")

	got, err := NewTaskRepository().ListUpcoming(context.Background(), db, s.WorkspaceID, s.MembershipID, 10)
	if err != nil {
		t.Fatalf("ListUpcoming() error = %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("ListUpcoming() = %#v, want empty non-nil slice", got)
	}
}

func TestTaskListUpcomingInvalidDueDate(t *testing.T) {
	db := newTestDB(t)
	s := seedWorkspace(t, db, "owner")
	p := seedProject(t, db, s.WorkspaceID, "p", baseNow)
	statuses := seedStatuses(t, db, s.WorkspaceID, p.ID)
	task := seedTask(t, db, s, p.ID, statuses[model.StatusToDo], "broken", "a0", nil)
	assign(t, db, s.WorkspaceID, task.ID, s.MembershipID)
	mustExec(t, db, `UPDATE tasks SET due_date = 'not-a-date' WHERE id = ?`, task.ID)

	_, err := NewTaskRepository().ListUpcoming(context.Background(), db, s.WorkspaceID, s.MembershipID, 10)
	if err == nil {
		t.Fatal("ListUpcoming() expected an error for an unparsable due date")
	}
}