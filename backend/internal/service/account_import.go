package service

import (
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	// MaxImportBytes is the largest archive that can be imported.
	MaxImportBytes = 2 << 30
	// maxImportDataBytes caps data.json once uncompressed.
	maxImportDataBytes = 256 << 20
)

// ImportResult says what an import did.
type ImportResult struct {
	// Restored is set when the workspace had no projects or resources
	// before: the import then also brought back the names and settings.
	Restored bool
	Added    ImportCounts
	// Existing counts rows that were still present and were kept as they are.
	Existing int
	// Renamed counts files that got a free name because the name was taken.
	Renamed int
	// MissingFiles lists files whose bytes were not in the archive; they
	// were left out.
	MissingFiles []string
}

type ImportCounts struct {
	Projects  int
	Plans     int
	Tasks     int
	Comments  int
	Resources int
	Labels    int
}

// importDocument is the part of data.json an import reads. Unknown tables
// and columns are ignored, so an export from a newer schema still imports.
type importDocument struct {
	Format  string `json:"format"`
	Version int    `json:"version"`
	Tables  struct {
		Users []struct {
			DisplayName string `json:"display_name"`
		} `json:"users"`
		Workspaces []struct {
			Name string `json:"name"`
		} `json:"workspaces"`
		Teams          []json.RawMessage          `json:"teams"`
		Projects       []model.ImportProject      `json:"projects"`
		Plans          []model.ImportPlan         `json:"plans"`
		Statuses       []model.ImportStatus       `json:"statuses"`
		Tasks          []model.ImportTask         `json:"tasks"`
		TaskAssignees  []model.ImportAssignee     `json:"task_assignees"`
		TaskComments   []model.ImportComment      `json:"task_comments"`
		Labels         []model.ImportLabel        `json:"labels"`
		TaskLabels     []model.ImportTaskLabel    `json:"task_labels"`
		Resources      []model.ImportResource     `json:"resources"`
		TaskResources  []model.ImportTaskResource `json:"task_resources"`
		ResourceEvents []model.ImportEvent        `json:"resource_events"`
		Repositories   []model.ImportRepository   `json:"resource_repositories"`
		UserSettings   []model.ImportSetting      `json:"user_settings"`
	} `json:"tables"`
}

// Import reads an archive made by Export and adds what it holds to the
// workspace. Nothing is deleted, and it is all or nothing.
//
// Rows keep their ids (see keepOrAdd): a row whose id is still live here is
// already present and kept as it is, a deleted one is brought back, so
// importing the same export twice adds nothing. A folder whose name is taken
// by a folder is merged into it; a file whose name is taken gets a free
// name. Links and
// assignments are only added for tasks or resources the import adds. When
// the workspace was empty, the user and workspace names and the settings
// are taken from the archive too.
func (s *AccountService) Import(ctx context.Context, session model.Session, body io.Reader) (ImportResult, error) {
	archive, err := s.store.Stage(ctx, io.LimitReader(body, MaxImportBytes+1))
	if err != nil {
		return ImportResult{}, err
	}
	defer s.store.Discard(archive)
	if archive.Size > MaxImportBytes {
		return ImportResult{}, fmt.Errorf("%w: archive", ErrTooLarge)
	}
	zr, err := zip.OpenReader(archive.TempPath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("%w: not a zip archive", ErrInvalidInput)
	}
	defer zr.Close()

	doc, err := readImportDocument(&zr.Reader)
	if err != nil {
		return ImportResult{}, err
	}
	if err := validateImport(&doc); err != nil {
		return ImportResult{}, err
	}

	var result ImportResult
	files, err := s.stageImportFiles(ctx, &zr.Reader, doc.Tables.Resources, &result)
	defer func() {
		for _, f := range files {
			s.store.Discard(f)
		}
	}()
	if err != nil {
		return ImportResult{}, err
	}

	imp := &importer{s: s, session: session, doc: &doc, files: files, result: &result,
		ids: map[string]string{}, added: map[string]bool{}}
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		return imp.run(ctx, tx)
	})
	if err != nil {
		// The rows are rolled back; take the bytes back off the disk too.
		for _, p := range slices.Backward(imp.placed) {
			_ = s.store.Remove(p) // best effort: nothing refers to them any more
		}
		if database.IsConstraint(err) {
			return ImportResult{}, fmt.Errorf("%w: inconsistent data: %v", ErrInvalidInput, err)
		}
		return ImportResult{}, err
	}
	return result, nil
}

func readImportDocument(zr *zip.Reader) (importDocument, error) {
	var doc importDocument
	for _, f := range zr.File {
		if f.Name != exportDataName {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return doc, fmt.Errorf("%w: unreadable %s", ErrInvalidInput, exportDataName)
		}
		defer rc.Close()
		data, err := io.ReadAll(io.LimitReader(rc, maxImportDataBytes+1))
		if err != nil {
			return doc, fmt.Errorf("%w: unreadable %s", ErrInvalidInput, exportDataName)
		}
		if len(data) > maxImportDataBytes {
			return doc, fmt.Errorf("%w: %s", ErrTooLarge, exportDataName)
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			return doc, fmt.Errorf("%w: malformed %s", ErrInvalidInput, exportDataName)
		}
		if doc.Format != ExportFormat || doc.Version < 1 || doc.Version > ExportVersion {
			return doc, fmt.Errorf("%w: not a supported export", ErrInvalidInput)
		}
		return doc, nil
	}
	return doc, fmt.Errorf("%w: %s missing", ErrInvalidInput, exportDataName)
}

// importPath gives an imported resource its path in the archive's tree.
// validateImport has made sure every chain ends at the top.
func importPath(byID map[string]*model.ImportResource, r *model.ImportResource) []string {
	path := []string{r.Name}
	for p := r.ParentID; p != nil; {
		parent := byID[*p]
		path = append([]string{parent.Name}, path...)
		p = parent.ParentID
	}
	return path
}

// stageImportFiles copies the bytes of the archive's files to temporary
// files before the transaction, keyed by the exported resource id.
func (s *AccountService) stageImportFiles(ctx context.Context, zr *zip.Reader, resources []model.ImportResource, result *ImportResult) (map[string]model.StagedFile, error) {
	entries := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		if _, ok := entries[f.Name]; !ok {
			entries[f.Name] = f
		}
	}
	byID := make(map[string]*model.ImportResource, len(resources))
	for i := range resources {
		byID[resources[i].ID] = &resources[i]
	}
	staged := map[string]model.StagedFile{}
	for i := range resources {
		r := &resources[i]
		if r.Kind != string(model.ResourceFile) {
			continue
		}
		path := strings.Join(importPath(byID, r), "/")
		entry, ok := entries[exportFilesDir+path]
		if !ok {
			result.MissingFiles = append(result.MissingFiles, path)
			continue
		}
		rc, err := entry.Open()
		if err != nil {
			return staged, fmt.Errorf("%w: unreadable file %s", ErrInvalidInput, path)
		}
		f, err := s.store.Stage(ctx, io.LimitReader(rc, MaxUploadBytes+1))
		rc.Close()
		if err != nil {
			if errors.Is(err, zip.ErrChecksum) || errors.Is(err, zip.ErrFormat) {
				return staged, fmt.Errorf("%w: damaged file %s", ErrInvalidInput, path)
			}
			return staged, err
		}
		staged[r.ID] = f
		if f.Size > MaxUploadBytes {
			return staged, fmt.Errorf("%w: file %s", ErrTooLarge, path)
		}
	}
	slices.Sort(result.MissingFiles)
	return staged, nil
}

// importer carries the state of one import transaction.
type importer struct {
	s       *AccountService
	session model.Session
	doc     *importDocument
	files   map[string]model.StagedFile
	result  *ImportResult
	now     string
	// ids maps exported ids to the ids used here: the same id for a row that
	// is still present, a new one for an added row.
	ids map[string]string
	// added holds the (local) ids of the rows this import inserted.
	added map[string]bool
	// taskProject maps the added tasks to their project, for their labels.
	taskProject map[string]string
	// idents are the live project identifiers, plus those of added projects.
	idents map[string]bool
	// placed are the disk paths written, removed again on failure.
	placed [][]string
}

// keepOrAdd maps an exported id and reports whether the row must be written.
// The id is kept unless another workspace (or the trash) has it: a live row
// is already here, a deleted one is brought back, a free id is used as it is.
// Keeping ids is what lets the same export be imported again without
// adding anything, also after a reset.
func (im *importer) keepOrAdd(ctx context.Context, tx *sql.Tx, table, oldID string) (string, bool, error) {
	state, err := im.s.repo.RowState(ctx, tx, im.session.WorkspaceID, table, oldID)
	if err != nil {
		return "", false, err
	}
	switch state {
	case model.RowLive:
		im.ids[oldID] = oldID
		im.result.Existing++
		return oldID, false, nil
	case model.RowElsewhere:
		id := model.NewID()
		im.ids[oldID] = id
		im.added[id] = true
		return id, true, nil
	default:
		im.ids[oldID] = oldID
		im.added[oldID] = true
		return oldID, true, nil
	}
}

func (im *importer) run(ctx context.Context, tx *sql.Tx) error {
	im.now = im.s.now().UTC().Format(time.RFC3339)
	hasContent, err := im.s.repo.HasContent(ctx, tx, im.session.WorkspaceID)
	if err != nil {
		return err
	}
	im.result.Restored = !hasContent

	steps := []func(context.Context, *sql.Tx) error{
		im.projects, im.plans, im.statuses, im.labels, im.tasks, im.taskLabels, im.assignees, im.comments,
		im.resources, im.repositories, im.taskResources, im.events,
	}
	if im.result.Restored {
		steps = append(steps, im.profile)
	}
	for _, step := range steps {
		if err := step(ctx, tx); err != nil {
			return err
		}
	}
	return nil
}

func (im *importer) projects(ctx context.Context, tx *sql.Tx) error {
	for _, p := range im.doc.Tables.Projects {
		id, add, err := im.keepOrAdd(ctx, tx, "projects", p.ID)
		if err != nil {
			return err
		}
		if !add {
			continue
		}
		p.ID = id
		if p.Identifier, err = im.identifierFor(ctx, tx, p); err != nil {
			return err
		}
		if err := im.s.repo.InsertProject(ctx, tx, im.session.WorkspaceID, p, im.now); err != nil {
			return err
		}
		im.result.Added.Projects++
	}
	return nil
}

// identifierFor keeps the exported identifier when it is valid and free in this
// workspace, and derives a new one from the name otherwise (an export made
// before identifiers existed has none).
func (im *importer) identifierFor(ctx context.Context, tx *sql.Tx, p model.ImportProject) (string, error) {
	if im.idents == nil {
		taken, err := im.s.repo.LiveIdentifiers(ctx, tx, im.session.WorkspaceID)
		if err != nil {
			return "", err
		}
		im.idents = taken
	}
	id := p.Identifier
	if !identifierPattern.MatchString(id) || im.idents[id] {
		id = suggestIdentifier(p.Name, im.idents)
	}
	im.idents[id] = true
	return id, nil
}

func (im *importer) plans(ctx context.Context, tx *sql.Tx) error {
	names := map[string]map[string]bool{} // taken plan names of kept projects
	for _, p := range im.doc.Tables.Plans {
		id, add, err := im.keepOrAdd(ctx, tx, "plans", p.ID)
		if err != nil {
			return err
		}
		if !add {
			continue
		}
		p.ID = id
		p.ProjectID = im.ids[p.ProjectID]
		if !im.added[p.ProjectID] {
			taken, ok := names[p.ProjectID]
			if !ok {
				if taken, err = im.s.repo.PlanNames(ctx, tx, im.session.WorkspaceID, p.ProjectID); err != nil {
					return err
				}
				names[p.ProjectID] = taken
			}
			p.Name = freePlanName(p.Name, taken)
			taken[p.Name] = true
		}
		if err := im.s.repo.InsertPlan(ctx, tx, im.session.WorkspaceID, p, im.now); err != nil {
			return err
		}
		im.result.Added.Plans++
	}
	return nil
}

// freePlanName returns name, or the first "name (n)" not taken.
func freePlanName(name string, taken map[string]bool) string {
	if !taken[name] {
		return name
	}
	for n := 2; ; n++ {
		candidate := name + " (" + strconv.Itoa(n) + ")"
		if !taken[candidate] {
			return candidate
		}
	}
}

func (im *importer) statuses(ctx context.Context, tx *sql.Tx) error {
	for _, st := range im.doc.Tables.Statuses {
		id, add, err := im.keepOrAdd(ctx, tx, "statuses", st.ID)
		if err != nil {
			return err
		}
		if !add {
			continue
		}
		st.ID = id
		st.PlanID = im.ids[st.PlanID]
		if err := im.s.repo.InsertStatus(ctx, tx, im.session.WorkspaceID, st, im.now); err != nil {
			return err
		}
	}
	return nil
}

func (im *importer) tasks(ctx context.Context, tx *sql.Tx) error {
	// Parents first: subtasks are one level deep.
	tasks := slices.Clone(im.doc.Tables.Tasks)
	slices.SortStableFunc(tasks, func(a, b model.ImportTask) int {
		return boolOrder(a.ParentID != nil) - boolOrder(b.ParentID != nil)
	})
	for _, t := range tasks {
		id, add, err := im.keepOrAdd(ctx, tx, "tasks", t.ID)
		if err != nil {
			return err
		}
		if !add {
			continue
		}
		t.ID = id
		// Exports made before priorities existed carry none.
		if t.Priority == "" {
			t.Priority = string(model.PriorityNone)
		}
		if !model.TaskPriority(t.Priority).Valid() {
			return fmt.Errorf("%w: task priority %q", ErrInvalidInput, t.Priority)
		}
		t.ProjectID = im.ids[t.ProjectID]
		t.PlanID = im.ids[t.PlanID]
		t.StatusID = im.ids[t.StatusID]
		if t.ParentID != nil {
			parent := im.ids[*t.ParentID]
			t.ParentID = &parent
		}
		// A task row that already exists (a deleted one coming back) keeps its
		// number; an added task takes the project's next one.
		number, err := im.s.repo.TaskNumber(ctx, tx, t.ID)
		if err != nil {
			return err
		}
		if number == 0 {
			if number, err = im.s.repo.ReserveTaskNumber(ctx, tx, im.session.WorkspaceID, t.ProjectID, im.now); err != nil {
				return err
			}
		}
		if err := im.s.repo.InsertTask(ctx, tx, im.session.WorkspaceID, im.session.MembershipID, t, number, im.now); err != nil {
			return err
		}
		if im.taskProject == nil {
			im.taskProject = map[string]string{}
		}
		im.taskProject[t.ID] = t.ProjectID
		im.result.Added.Tasks++
	}
	return nil
}

// labels adds the exported labels. A label whose name a kept project already
// uses is not added again: its tasks get the existing label.
func (im *importer) labels(ctx context.Context, tx *sql.Tx) error {
	for _, l := range im.doc.Tables.Labels {
		id, add, err := im.keepOrAdd(ctx, tx, "labels", l.ID)
		if err != nil {
			return err
		}
		if !add {
			continue
		}
		name, err := normalizeLabelName(l.Name)
		if err != nil {
			return err
		}
		if !validLabelColor(l.Color) {
			return fmt.Errorf("%w: label color %q", ErrInvalidInput, l.Color)
		}
		l.ID, l.Name = id, name
		l.ProjectID = im.ids[l.ProjectID]
		if !im.added[l.ProjectID] {
			existing, found, err := im.s.repo.LabelIDByName(ctx, tx, im.session.WorkspaceID, l.ProjectID, l.Name)
			if err != nil {
				return err
			}
			if found {
				im.ids[l.ID] = existing
				delete(im.added, id)
				im.ids[id] = existing
				continue
			}
		}
		if err := im.s.repo.InsertLabel(ctx, tx, im.session.WorkspaceID, l, im.now); err != nil {
			return err
		}
		im.result.Added.Labels++
	}
	return nil
}

// taskLabels labels the tasks this import added.
func (im *importer) taskLabels(ctx context.Context, tx *sql.Tx) error {
	for _, l := range im.doc.Tables.TaskLabels {
		taskID, okTask := im.ids[l.TaskID]
		labelID, okLabel := im.ids[l.LabelID]
		if !okTask || !okLabel || !im.added[taskID] {
			continue
		}
		l.TaskID, l.LabelID = taskID, labelID
		if err := im.s.repo.InsertTaskLabel(ctx, tx, im.session.WorkspaceID, im.taskProject[taskID], l, im.now); err != nil {
			return err
		}
	}
	return nil
}

func boolOrder(b bool) int {
	if b {
		return 1
	}
	return 0
}

// assignees assigns the added tasks to the importing member: a personal
// workspace has no one else.
func (im *importer) assignees(ctx context.Context, tx *sql.Tx) error {
	for _, a := range im.doc.Tables.TaskAssignees {
		taskID := im.ids[a.TaskID]
		if !im.added[taskID] {
			continue
		}
		if err := im.s.repo.InsertAssignee(ctx, tx, im.session.WorkspaceID, taskID, im.session.MembershipID, a.AssignedAt, im.now); err != nil {
			return err
		}
	}
	return nil
}

func (im *importer) comments(ctx context.Context, tx *sql.Tx) error {
	for _, c := range im.doc.Tables.TaskComments {
		id, add, err := im.keepOrAdd(ctx, tx, "task_comments", c.ID)
		if err != nil {
			return err
		}
		if !add {
			continue
		}
		c.ID = id
		c.TaskID = im.ids[c.TaskID]
		if err := im.s.repo.InsertComment(ctx, tx, im.session.WorkspaceID, im.session.MembershipID, c); err != nil {
			return err
		}
		im.result.Added.Comments++
	}
	return nil
}

// localTree is the live resource tree of the workspace, kept up to date as
// the import adds to it.
type localTree struct {
	parent map[string]string
	name   map[string]string
	kind   map[string]model.ResourceKind
	// repo marks repository folders, which nothing may be merged into.
	repo map[string]bool
	// children maps a folder id ("" for the top) to its children's name keys
	// and ids, the shape freeName wants.
	children map[string]map[string]string
}

func newLocalTree(resources []model.Resource) *localTree {
	t := &localTree{parent: map[string]string{}, name: map[string]string{},
		kind: map[string]model.ResourceKind{}, repo: map[string]bool{}, children: map[string]map[string]string{}}
	for _, r := range resources {
		t.add(r.ID, r.ParentID, r.Name, r.Kind)
		t.repo[r.ID] = r.IsRepository
	}
	return t
}

func (t *localTree) add(id, parentID, name string, kind model.ResourceKind) {
	t.parent[id], t.name[id], t.kind[id] = parentID, name, kind
	if t.children[parentID] == nil {
		t.children[parentID] = map[string]string{}
	}
	t.children[parentID][nameKey(name)] = id
}

func (t *localTree) path(workspaceID, id string) []string {
	var names []string
	for cur := id; cur != ""; cur = t.parent[cur] {
		names = append([]string{t.name[cur]}, names...)
	}
	return append([]string{workspaceID}, names...)
}

func (im *importer) resources(ctx context.Context, tx *sql.Tx) error {
	current, err := im.s.resources.ListLive(ctx, tx, im.session.WorkspaceID)
	if err != nil {
		return err
	}
	tree := newLocalTree(current)

	resources := slices.Clone(im.doc.Tables.Resources)
	byID := make(map[string]*model.ImportResource, len(resources))
	for i := range resources {
		byID[resources[i].ID] = &resources[i]
	}
	depth := make(map[string]int, len(resources))
	for i := range resources {
		depth[resources[i].ID] = len(importPath(byID, &resources[i]))
	}
	slices.SortStableFunc(resources, func(a, b model.ImportResource) int { return depth[a.ID] - depth[b.ID] })
	repoFolders := make(map[string]bool, len(im.doc.Tables.Repositories))
	for _, rr := range im.doc.Tables.Repositories {
		repoFolders[rr.ResourceID] = true
	}

	for _, r := range resources {
		state, err := im.s.repo.RowState(ctx, tx, im.session.WorkspaceID, "resources", r.ID)
		if err != nil {
			return err
		}
		if state == model.RowLive {
			im.ids[r.ID] = r.ID
			im.result.Existing++
			continue
		}
		parentID := ""
		if r.ParentID != nil {
			var ok bool
			if parentID, ok = im.ids[*r.ParentID]; !ok {
				continue // cannot happen: folders are always placed
			}
		}
		kind := model.ResourceKind(r.Kind)
		staged, hasBytes := im.files[r.ID]
		if kind == model.ResourceFile && !hasBytes {
			continue // listed in MissingFiles
		}
		siblings := tree.children[parentID]
		if existing, taken := siblings[nameKey(r.Name)]; taken {
			if kind == model.ResourceFolder && tree.kind[existing] == model.ResourceFolder &&
				!tree.repo[existing] && !repoFolders[r.ID] {
				im.ids[r.ID] = existing
				im.result.Existing++
				continue
			}
			r.Name = freeName(r.Name, kind, siblings)
			im.result.Renamed++
		}

		oldID := r.ID
		if state == model.RowElsewhere {
			r.ID = model.NewID()
		}
		im.ids[oldID] = r.ID
		im.added[r.ID] = true
		r.ParentID = nil
		if parentID != "" {
			r.ParentID = &parentID
		}
		if kind == model.ResourceFile {
			r.Size = staged.Size
			r.IsText = staged.IsText && staged.Size <= MaxTextBytes
			if r.MimeType == nil || *r.MimeType == "" {
				m := detectMime(r.Name, "", staged.Head)
				r.MimeType = &m
			}
		} else {
			r.MimeType, r.Size, r.IsText = nil, 0, false
		}
		if err := im.s.repo.InsertResource(ctx, tx, im.session.WorkspaceID, im.session.MembershipID, nameKey(r.Name), r, im.now); err != nil {
			return err
		}
		tree.add(r.ID, parentID, r.Name, kind)

		path := tree.path(im.session.WorkspaceID, r.ID)
		if kind == model.ResourceFolder {
			err = im.s.store.MakeDir(path)
		} else {
			err = im.s.store.Place(staged, path)
		}
		if err != nil {
			return err
		}
		im.placed = append(im.placed, path)
		im.result.Added.Resources++
	}
	return nil
}

// repositories makes the folders the import added repository folders again.
// Their files are fetched afterwards, like those of a newly added repository.
func (im *importer) repositories(ctx context.Context, tx *sql.Tx) error {
	for _, rr := range im.doc.Tables.Repositories {
		resourceID, ok := im.ids[rr.ResourceID]
		if !ok || !im.added[resourceID] {
			continue
		}
		rr.ResourceID = resourceID
		if err := im.s.repo.InsertRepository(ctx, tx, im.session.WorkspaceID, im.session.MembershipID, rr, im.now); err != nil {
			return err
		}
	}
	return nil
}

func (im *importer) taskResources(ctx context.Context, tx *sql.Tx) error {
	for _, l := range im.doc.Tables.TaskResources {
		taskID, okTask := im.ids[l.TaskID]
		resourceID, okRes := im.ids[l.ResourceID]
		if !okTask || !okRes || !(im.added[taskID] || im.added[resourceID]) {
			continue
		}
		l.TaskID, l.ResourceID = taskID, resourceID
		if err := im.s.repo.InsertTaskResource(ctx, tx, im.session.WorkspaceID, im.session.MembershipID, l, im.now); err != nil {
			return err
		}
	}
	return nil
}

// events brings along the history of the resources the import added. Ids
// that have to change carry the original times, so the log keeps its order.
func (im *importer) events(ctx context.Context, tx *sql.Tx) error {
	// Exported in id order, which is the order they happened in.
	var last time.Time
	for _, e := range im.doc.Tables.ResourceEvents {
		resourceID := im.ids[e.ResourceID]
		if !im.added[resourceID] {
			continue
		}
		id, add, err := im.keepOrAdd(ctx, tx, "resource_events", e.ID)
		if err != nil {
			return err
		}
		if !add {
			continue
		}
		if id != e.ID {
			// A new id must sort where the old one did. Times have whole
			// seconds; events of the same second keep their exported order
			// by stepping a millisecond.
			created, _ := time.Parse(time.RFC3339, e.CreatedAt) // checked by validateImport
			if !created.After(last) {
				created = last.Add(time.Millisecond)
			}
			last = created
			id = model.NewIDAt(created)
			im.ids[e.ID] = id
		}
		e.ID = id
		e.ResourceID = resourceID
		if e.TaskID != nil {
			if taskID, ok := im.ids[*e.TaskID]; ok {
				e.TaskID = &taskID
			} else {
				e.TaskID = nil
			}
		}
		if err := im.s.repo.InsertEvent(ctx, tx, im.session.WorkspaceID, im.session.MembershipID, e); err != nil {
			return err
		}
	}
	return nil
}

// profile takes the names and settings from the archive; only done when the
// workspace was empty, so a merge never overwrites them.
func (im *importer) profile(ctx context.Context, tx *sql.Tx) error {
	if users := im.doc.Tables.Users; len(users) > 0 {
		if err := im.s.workspaces.UpdateDisplayName(ctx, tx, im.session.UserID, users[0].DisplayName, im.now); err != nil {
			return err
		}
	}
	if ws := im.doc.Tables.Workspaces; len(ws) > 0 {
		if err := im.s.workspaces.UpdateName(ctx, tx, im.session.WorkspaceID, ws[0].Name, im.now); err != nil {
			return err
		}
	}
	for _, st := range im.doc.Tables.UserSettings {
		if err := im.s.settings.Put(ctx, tx, im.session.UserID, st.Key, st.Value, im.now); err != nil {
			return err
		}
	}
	return nil
}

// validateImport checks and normalizes the rows the way the services do for
// rows made in the app, and checks that every reference points to a row of
// the archive. The database constraints stay as the last line.
func validateImport(doc *importDocument) error {
	tb := &doc.Tables
	invalid := func(what string) error { return fmt.Errorf("%w: %s", ErrInvalidInput, what) }

	if len(tb.Teams) > 0 {
		return invalid("teams are not supported in a personal workspace")
	}
	for i := range tb.Users {
		name, err := normalizeProfileName(&tb.Users[i].DisplayName, "user name")
		if err != nil {
			return err
		}
		tb.Users[i].DisplayName = name
	}
	for i := range tb.Workspaces {
		name, err := normalizeProfileName(&tb.Workspaces[i].Name, "workspace name")
		if err != nil {
			return err
		}
		tb.Workspaces[i].Name = name
	}

	seen := map[string]bool{}
	unique := func(id, what string) error {
		if id == "" || seen[id] {
			return invalid(what + " id")
		}
		seen[id] = true
		return nil
	}

	projects := map[string]bool{}
	for i := range tb.Projects {
		p := &tb.Projects[i]
		if err := unique(p.ID, "project"); err != nil {
			return err
		}
		var err error
		if p.Name, err = requiredText(p.Name, "project name"); err != nil {
			return err
		}
		if p.Icon, err = optionalIcon(p.Icon); err != nil {
			return err
		}
		if err := checkTimes(p.CreatedAt, p.ArchivedAt); err != nil {
			return err
		}
		projects[p.ID] = true
	}

	plans := map[string]model.ImportPlan{}
	projectHasPlan := map[string]bool{}
	for i := range tb.Plans {
		p := &tb.Plans[i]
		if err := unique(p.ID, "plan"); err != nil {
			return err
		}
		if p.TeamID != nil {
			return invalid("team plans are not supported in a personal workspace")
		}
		if !projects[p.ProjectID] {
			return invalid("plan without its project")
		}
		var err error
		if p.Name, err = requiredText(p.Name, "plan name"); err != nil {
			return err
		}
		if p.Icon, err = optionalIcon(p.Icon); err != nil {
			return err
		}
		if err := checkTimes(p.CreatedAt, p.ArchivedAt); err != nil {
			return err
		}
		plans[p.ID] = *p
		if p.ArchivedAt == nil {
			projectHasPlan[p.ProjectID] = true
		}
	}
	for id := range projects {
		if !projectHasPlan[id] {
			return invalid("project without a whole-project plan")
		}
	}

	statusPlan := map[string]string{}
	planHasDone := map[string]bool{}
	for i := range tb.Statuses {
		st := &tb.Statuses[i]
		if err := unique(st.ID, "status"); err != nil {
			return err
		}
		if _, ok := plans[st.PlanID]; !ok {
			return invalid("status without its plan")
		}
		var err error
		if st.Title, err = requiredText(st.Title, "status title"); err != nil {
			return err
		}
		if !colorPattern.MatchString(st.Color) {
			return invalid("status color")
		}
		st.Color = strings.ToLower(st.Color)
		switch model.StatusCategory(st.Category) {
		case model.StatusToDo, model.StatusInProgress:
		case model.StatusDone:
			planHasDone[st.PlanID] = true
		default:
			return invalid("status category")
		}
		if st.Position == "" {
			return invalid("status position")
		}
		statusPlan[st.ID] = st.PlanID
	}
	for id := range plans {
		if !planHasDone[id] {
			return invalid("plan without a done status")
		}
	}

	tasks := map[string]model.ImportTask{}
	for i := range tb.Tasks {
		t := &tb.Tasks[i]
		if err := unique(t.ID, "task"); err != nil {
			return err
		}
		plan, ok := plans[t.PlanID]
		if !ok || plan.ProjectID != t.ProjectID || statusPlan[t.StatusID] != t.PlanID {
			return invalid("task outside its plan or status")
		}
		var err error
		if t.Title, err = requiredText(t.Title, "task title"); err != nil {
			return err
		}
		if t.Position == "" {
			return invalid("task position")
		}
		if err := checkTimes(t.CreatedAt, t.DueDate, t.CompletedAt); err != nil {
			return err
		}
		tasks[t.ID] = *t
	}
	for _, t := range tb.Tasks {
		if t.ParentID == nil {
			continue
		}
		parent, ok := tasks[*t.ParentID]
		if !ok || parent.ParentID != nil || parent.ProjectID != t.ProjectID {
			return invalid("subtask without a top-level parent in its project")
		}
	}

	for _, a := range tb.TaskAssignees {
		if _, ok := tasks[a.TaskID]; !ok {
			return invalid("assignment without its task")
		}
		if err := checkTimes(a.AssignedAt); err != nil {
			return err
		}
	}

	for i := range tb.TaskComments {
		c := &tb.TaskComments[i]
		if err := unique(c.ID, "comment"); err != nil {
			return err
		}
		if _, ok := tasks[c.TaskID]; !ok {
			return invalid("comment without its task")
		}
		var err error
		if c.Body, err = normalizeCommentBody(c.Body); err != nil {
			return err
		}
		if err := checkTimes(c.CreatedAt, &c.UpdatedAt); err != nil {
			return err
		}
	}

	resources := map[string]*model.ImportResource{}
	for i := range tb.Resources {
		r := &tb.Resources[i]
		if err := unique(r.ID, "resource"); err != nil {
			return err
		}
		resources[r.ID] = r
	}
	for _, r := range resources {
		switch model.ResourceKind(r.Kind) {
		case model.ResourceFolder, model.ResourceFile:
		default:
			return invalid("resource kind")
		}
		parentID := ""
		if r.ParentID != nil {
			parent, ok := resources[*r.ParentID]
			if !ok || parent.Kind != string(model.ResourceFolder) {
				return invalid("resource without its folder")
			}
			parentID = parent.ID
		}
		name, err := normalizeResourceName(r.Name)
		if err != nil {
			return err
		}
		if err := checkReservedName(parentID, name); err != nil {
			return err
		}
		r.Name = name
		if r.Revision < 1 {
			r.Revision = 1
		}
		if err := checkTimes(r.CreatedAt); err != nil {
			return err
		}
	}
	// Every parent is known now. A chain longer than the number of
	// resources loops.
	for _, r := range resources {
		steps := 0
		for p := r.ParentID; p != nil; p = resources[*p].ParentID {
			if steps++; steps > len(resources) {
				return invalid("resource folders form a cycle")
			}
		}
	}
	// Sibling names must be unique, as in the tree they came from.
	siblings := map[string]bool{}
	for _, r := range resources {
		key := ""
		if r.ParentID != nil {
			key = *r.ParentID
		}
		key += "/" + nameKey(r.Name)
		if siblings[key] {
			return invalid("two resources with the same name in one folder")
		}
		siblings[key] = true
	}
	repoFolders := map[string]bool{}
	for _, rr := range tb.Repositories {
		folder, ok := resources[rr.ResourceID]
		switch {
		case !ok || folder.Kind != string(model.ResourceFolder):
			return invalid("repository without its folder")
		case repoFolders[rr.ResourceID]:
			return invalid("two repositories in one folder")
		case rr.GitHubID <= 0, len(rr.FullName) < 3, len(rr.FullName) > 200, !strings.Contains(rr.FullName, "/"),
			!validBranch(rr.Branch), rr.IsPrivate != 0 && rr.IsPrivate != 1:
			return invalid("repository")
		}
		if !strings.HasPrefix(rr.HTMLURL, "https://") {
			return invalid("repository url")
		}
		repoFolders[rr.ResourceID] = true
	}
	for _, r := range resources {
		if r.ParentID != nil && repoFolders[*r.ParentID] {
			return invalid("resource inside a repository")
		}
	}

	for _, l := range tb.TaskResources {
		if _, ok := tasks[l.TaskID]; !ok {
			return invalid("link without its task")
		}
		if _, ok := resources[l.ResourceID]; !ok {
			return invalid("link without its resource")
		}
		if err := checkTimes(l.LinkedAt); err != nil {
			return err
		}
	}

	for i := range tb.ResourceEvents {
		e := &tb.ResourceEvents[i]
		if _, ok := resources[e.ResourceID]; !ok {
			// History of resources that were not exported (trashed or
			// purged) has nothing to attach to.
			continue
		}
		if !validAction(model.ResourceAction(e.Action)) {
			return invalid("resource event action")
		}
		var fields map[string]any
		if json.Unmarshal([]byte(e.Details), &fields) != nil || fields == nil {
			e.Details = "{}"
		}
		if err := checkTimes(e.CreatedAt, &e.UpdatedAt); err != nil {
			return err
		}
	}

	for _, st := range tb.UserSettings {
		if st.Key == "" || !json.Valid([]byte(st.Value)) {
			return invalid("setting")
		}
	}
	return nil
}

func requiredText(s, what string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("%w: %s is required", ErrInvalidInput, what)
	}
	return s, nil
}

func optionalIcon(icon *string) (*string, error) {
	if icon == nil {
		return nil, nil
	}
	v, err := normalizeIcon(*icon)
	if err != nil || v == "" {
		return nil, err
	}
	return &v, nil
}

// checkTimes accepts a required time followed by optional ones, all RFC 3339.
func checkTimes(required string, optional ...*string) error {
	if _, err := time.Parse(time.RFC3339, required); err != nil {
		return fmt.Errorf("%w: time %q", ErrInvalidInput, required)
	}
	for _, t := range optional {
		if t == nil {
			continue
		}
		if _, err := time.Parse(time.RFC3339, *t); err != nil {
			return fmt.Errorf("%w: time %q", ErrInvalidInput, *t)
		}
	}
	return nil
}

func validAction(a model.ResourceAction) bool {
	switch a {
	case model.ResourceCreated, model.ResourceUploaded, model.ResourceEdited, model.ResourceRenamed,
		model.ResourceMoved, model.ResourceCopied, model.ResourceTrashed, model.ResourceRestored,
		model.ResourcePurged, model.ResourceLinked, model.ResourceUnlinked:
		return true
	}
	return false
}
