package database

import "testing"

func TestTasks_CompositeForeignKeys(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)

	cases := []struct {
		name                           string
		team, project, status, creator int64
		parent                         any
		wantErr                        bool
	}{
		{"tutarlı kayıt", f.teamA, f.projA, f.todoA, f.memA, nil, false},
		{"aynı projede alt görev", f.teamA, f.projA, f.todoA, f.memA, f.taskA, false},
		{"proje başka takıma ait", f.teamA, f.projB, f.todoB, f.memA, nil, true},
		{"status başka projeye ait", f.teamA, f.projA, f.todoB, f.memA, nil, true},
		{"oluşturan başka takımın üyesi", f.teamA, f.projA, f.todoA, f.memB, nil, true},
		{"üst görev başka projede", f.teamA, f.projA, f.todoA, f.memA, f.taskB, true},
		{"var olmayan proje", f.teamA, 9999, f.todoA, f.memA, nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec(
				`INSERT INTO tasks (team_id, project_id, status_id, parent_id, created_by, title)
				 VALUES (?, ?, ?, ?, ?, 'x')`,
				tc.team, tc.project, tc.status, tc.parent, tc.creator)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestTaskAssignees_CompositeForeignKeys(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)

	cases := []struct {
		name                string
		task, member, team  int64
		wantErr             bool
	}{
		{"tutarlı atama A", f.taskA, f.memA, f.teamA, false},
		{"tutarlı atama B", f.taskB, f.memB, f.teamB, false},
		{"üye başka takımın", f.taskA, f.memB, f.teamA, true},
		{"team_id göreve uymuyor", f.taskA, f.memA, f.teamB, true},
		{"ikisi de yanlış takımda", f.taskB, f.memA, f.teamB, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec(
				`INSERT INTO task_assignees (task_id, membership_id, team_id) VALUES (?, ?, ?)`,
				tc.task, tc.member, tc.team)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestTaskAssignees_DuplicateRejected(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)
	q := `INSERT INTO task_assignees (task_id, membership_id, team_id) VALUES (?, ?, ?)`

	if _, err := db.Exec(q, f.taskA, f.memA, f.teamA); err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if _, err := db.Exec(q, f.taskA, f.memA, f.teamA); err == nil {
		t.Fatal("duplicate assignment accepted")
	}
}

func TestConstraints(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)

	cases := []struct {
		name    string
		query   string
		args    []any
		wantErr bool
	}{
		{"geçersiz rol",
			`INSERT INTO memberships (team_id, account_id, role) VALUES (?, ?, 'god')`,
			[]any{f.teamA, f.accB}, true},
		{"geçerli rol",
			`INSERT INTO memberships (team_id, account_id, role) VALUES (?, ?, 'member')`,
			[]any{f.teamA, f.accB}, false},
		{"aynı hesap aynı takıma ikinci kez",
			`INSERT INTO memberships (team_id, account_id, role) VALUES (?, ?, 'member')`,
			[]any{f.teamA, f.accA}, true},
		{"e-posta büyük/küçük harf farkıyla tekrar",
			`INSERT INTO accounts (email, password_hash, display_name) VALUES ('A@X.COM', 'h', 'dup')`,
			nil, true},
		{"e-posta çok kısa",
			`INSERT INTO accounts (email, password_hash, display_name) VALUES ('a@', 'h', 'short')`,
			nil, true},
		{"aynı takımda aynı proje adı",
			`INSERT INTO projects (team_id, name) VALUES (?, 'Alpha')`,
			[]any{f.teamA}, true},
		{"başka takımda aynı proje adı serbest",
			`INSERT INTO projects (team_id, name) VALUES (?, 'Beta')`,
			[]any{f.teamB}, false},
		{"renk kodu hex değil",
			`INSERT INTO statuses (project_id, title, color, category, position) VALUES (?, 's1', 'red', 'todo', 9)`,
			[]any{f.projA}, true},
		{"renk kodunda geçersiz karakter",
			`INSERT INTO statuses (project_id, title, color, category, position) VALUES (?, 's2', '#12345g', 'todo', 9)`,
			[]any{f.projA}, true},
		{"bilinmeyen kategori",
			`INSERT INTO statuses (project_id, title, color, category, position) VALUES (?, 's3', '#123456', 'doing', 9)`,
			[]any{f.projA}, true},
		{"projede aynı status başlığı",
			`INSERT INTO statuses (project_id, title, color, category, position) VALUES (?, 'Todo', '#123456', 'todo', 9)`,
			[]any{f.projA}, true},
		{"boş görev başlığı",
			`INSERT INTO tasks (team_id, project_id, status_id, created_by, title) VALUES (?, ?, ?, ?, '')`,
			[]any{f.teamA, f.projA, f.todoA, f.memA}, true},
		{"görev kendi üst görevi",
			`UPDATE tasks SET parent_id = id WHERE id = ?`,
			[]any{f.taskA}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := db.Exec(tc.query, tc.args...)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestDelete_UsedStatusIsRejected(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)

	if _, err := db.Exec(`DELETE FROM statuses WHERE id = ?`, f.todoA); err == nil {
		t.Fatal("görevin kullandığı status silindi")
	}
	if _, err := db.Exec(`DELETE FROM statuses WHERE id = ?`, f.doneA); err != nil {
		t.Fatalf("kullanılmayan status silinemedi: %v", err)
	}
}

func TestDelete_Cascades(t *testing.T) {
	type check struct {
		table, where string
		arg          int64
	}
	cases := []struct {
		name   string
		del    string
		arg    func(fixture) int64
		wantGone func(fixture) []check
	}{
		{
			name: "proje silinince status, görev ve atamalar gider",
			del:  `DELETE FROM projects WHERE id = ?`,
			arg:  func(f fixture) int64 { return f.projA },
			wantGone: func(f fixture) []check {
				return []check{
					{"statuses", "project_id = ?", f.projA},
					{"tasks", "project_id = ?", f.projA},
					{"task_assignees", "task_id = ?", f.taskA},
				}
			},
		},
		{
			name: "takım silinince üyelik, proje, görev ve atamalar gider",
			del:  `DELETE FROM teams WHERE id = ?`,
			arg:  func(f fixture) int64 { return f.teamA },
			wantGone: func(f fixture) []check {
				return []check{
					{"memberships", "team_id = ?", f.teamA},
					{"projects", "team_id = ?", f.teamA},
					{"tasks", "team_id = ?", f.teamA},
					{"task_assignees", "team_id = ?", f.teamA},
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newTestDB(t)
			f := seed(t, db)
			insertID(t, db,
				`INSERT INTO task_assignees (task_id, membership_id, team_id) VALUES (?, ?, ?)`,
				f.taskA, f.memA, f.teamA)

			if _, err := db.Exec(tc.del, tc.arg(f)); err != nil {
				t.Fatalf("delete: %v", err)
			}
			for _, c := range tc.wantGone(f) {
				if n := count(t, db, c.table, c.where, c.arg); n != 0 {
					t.Errorf("%s: %d satır kaldı", c.table, n)
				}
			}
			if n := count(t, db, "tasks", "project_id = ?", f.projB); n != 1 {
				t.Errorf("diğer takımın görevi etkilendi: %d", n)
			}
		})
	}
}

func TestTasks_ClosedAtOnInsert(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)

	cases := []struct {
		name       string
		status     int64
		wantClosed bool
	}{
		{"todo", f.todoA, false},
		{"done", f.doneA, true},
		{"cancelled", f.cancelA, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id := newTask(t, db, f, tc.status)
			if got := closedAt(t, db, id).Valid; got != tc.wantClosed {
				t.Errorf("closed_at set=%v, want %v", got, tc.wantClosed)
			}
		})
	}
}

func TestTasks_ClosedAtFollowsStatusTransitions(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)
	id := newTask(t, db, f, f.todoA)

	steps := []struct {
		name       string
		to         int64
		wantClosed bool
	}{
		{"todo -> done", f.doneA, true},
		{"done -> todo (yeniden açma)", f.todoA, false},
		{"todo -> cancelled", f.cancelA, true},
		{"cancelled -> done", f.doneA, true},
		{"done -> todo", f.todoA, false},
	}
	for _, s := range steps {
		if _, err := db.Exec(`UPDATE tasks SET updated_at = 0 WHERE id = ?`, id); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`UPDATE tasks SET status_id = ? WHERE id = ?`, s.to, id); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		if got := closedAt(t, db, id).Valid; got != s.wantClosed {
			t.Errorf("%s: closed_at set=%v, want %v", s.name, got, s.wantClosed)
		}
		if n := count(t, db, "tasks", "id = ? AND updated_at > 0", id); n != 1 {
			t.Errorf("%s: updated_at güncellenmedi", s.name)
		}
	}
}

func TestTasks_TitleUpdateDoesNotTouchClosedAt(t *testing.T) {
	db := newTestDB(t)
	f := seed(t, db)
	id := newTask(t, db, f, f.doneA)
	before := closedAt(t, db, id)

	if _, err := db.Exec(`UPDATE tasks SET closed_at = 1, title = 'renamed' WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	if got := closedAt(t, db, id); got.Int64 != 1 {
		t.Errorf("closed_at=%v, trigger status dışı güncellemede tetiklendi (önceki: %v)", got, before)
	}
}