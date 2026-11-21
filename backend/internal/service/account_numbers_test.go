package service

import "testing"

type keyRow struct {
	identifier, task string
	number           int
}

func projectKeys(t *testing.T, e accountEnv) []keyRow {
	t.Helper()
	rows, err := e.db.Query(`SELECT p.identifier, t.title, t.number FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE t.deleted_at IS NULL AND p.deleted_at IS NULL ORDER BY p.identifier, t.number`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []keyRow
	for rows.Next() {
		var k keyRow
		if err := rows.Scan(&k.identifier, &k.task, &k.number); err != nil {
			t.Fatal(err)
		}
		out = append(out, k)
	}
	return out
}

func TestImportKeepsKeysAfterReset(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	before := projectKeys(t, e)
	archive := e.exportBytes(t)
	if err := e.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	e.importBytes(t, archive)

	after := projectKeys(t, e)
	if len(before) != 1 || len(after) != 1 || after[0] != before[0] || after[0].identifier != "KEP" || after[0].number != 1 {
		t.Errorf("keys after import = %+v, before %+v; want KEP-1 back", after, before)
	}

	// The restored project goes on counting.
	var projectID string
	if err := e.db.QueryRow(`SELECT id FROM projects WHERE deleted_at IS NULL`).Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	next, err := e.tasks.Create(e.ctx, e.session, projectID, CreateTaskInput{Title: "after"})
	if err != nil || next.Number != 2 {
		t.Errorf("next number = %d (err %v), want 2", next.Number, err)
	}
}

func TestImportOfOlderExportGetsKeys(t *testing.T) {
	e := newAccountEnv(t)
	e.seed(t)
	archive := rewrite(t, e.exportBytes(t), func(doc map[string]any) {
		for _, row := range tableRows(doc, "projects") {
			delete(row.(map[string]any), "identifier")
		}
		for _, row := range tableRows(doc, "tasks") {
			delete(row.(map[string]any), "number")
		}
	})
	if err := e.svc.Reset(e.ctx, e.session); err != nil {
		t.Fatal(err)
	}
	e.importBytes(t, archive)

	got := projectKeys(t, e)
	if len(got) != 1 || got[0].identifier != "KEP" || got[0].number == 0 {
		t.Errorf("keys = %+v, want a KEP identifier and a number", got)
	}
}
