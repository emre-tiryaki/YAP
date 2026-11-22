package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/llm"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	aiSystemPrompt = "You are the assistant built into YAP, a personal project planner. The user may attach tasks, plans, projects and files to a message; they appear in <attachment> tags before the message text. Use them to answer. You cannot change anything in YAP: you cannot create, edit, move or delete tasks, plans, projects or files, so never claim you did; instead tell the user what they could do. Reply in the language the user writes in. Use Markdown."

	// aiAttachmentLimit caps the text of one attachment, aiTotalLimit the
	// attachment text of one request. Below aiMinBudget an attachment is
	// left out instead of cut.
	aiAttachmentLimit = 64 << 10
	aiTotalLimit      = 256 << 10
	aiMinBudget       = 4 << 10

	aiTruncatedMarker  = "\n[truncated]"
	aiMaxComments      = 20
	aiPlanLineDescRune = 200
)

type aiTaskSource interface {
	Get(ctx context.Context, q database.DBTX, workspaceID, taskID string) (model.Task, error)
	ListByPlan(ctx context.Context, q database.DBTX, workspaceID, planID string) ([]model.Task, error)
	ListSubtasks(ctx context.Context, q database.DBTX, workspaceID, parentID string) ([]model.Task, error)
	LabelIDsByTask(ctx context.Context, q database.DBTX, workspaceID string, taskIDs []string) (map[string][]string, error)
}

type aiActivitySource interface {
	StatusTitle(ctx context.Context, q database.DBTX, workspaceID, statusID string) (string, error)
	LabelNames(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]string, error)
}

type aiCommentSource interface {
	ListByTask(ctx context.Context, q database.DBTX, workspaceID, taskID string) ([]model.Comment, error)
}

type aiProjectSource interface {
	Get(ctx context.Context, q database.DBTX, workspaceID, projectID string) (model.Project, error)
	GetPlan(ctx context.Context, q database.DBTX, workspaceID, projectID, planID string) (model.Plan, error)
	ListPlans(ctx context.Context, q database.DBTX, workspaceID, projectID string) ([]model.Plan, error)
	ListStatuses(ctx context.Context, q database.DBTX, workspaceID, planID string) ([]model.Status, error)
}

type aiResourceSource interface {
	Get(ctx context.Context, session model.Session, id string) (model.Resource, error)
	Open(ctx context.Context, session model.Session, id string) (model.Resource, *os.File, error)
	TaskResources(ctx context.Context, session model.Session, taskID string) ([]model.Resource, error)
}

type aiRepoSource interface {
	Get(ctx context.Context, session model.Session, resourceID string) (model.Repository, error)
	OpenFile(ctx context.Context, session model.Session, resourceID, filePath string) (model.RepoEntry, *os.File, error)
}

type aiPlacementSource interface {
	LivePlanProjects(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]string, error)
	LiveProjectIDs(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]bool, error)
}

// AIContextSources are what the assistant reads attachments from.
type AIContextSources struct {
	Tasks     aiTaskSource
	Activity  aiActivitySource
	Comments  aiCommentSource
	Projects  aiProjectSource
	Resources aiResourceSource
	Repos     aiRepoSource
	Placement aiPlacementSource
}

// aiPrompt turns chats and their attachments into provider requests.
type aiPrompt struct {
	db  *sql.DB
	src AIContextSources
}

func newAIPrompt(db *sql.DB, src AIContextSources) *aiPrompt {
	return &aiPrompt{db: db, src: src}
}

func aiSystem(instructions string) string {
	instructions = strings.TrimSpace(instructions)
	if instructions == "" {
		return aiSystemPrompt
	}
	return aiSystemPrompt + "\n\nThe user's personal instructions:\n<instructions>\n" + instructions + "\n</instructions>"
}

// rendered is an attachment before the budget is applied. The body is what
// counts against the budget; open and close frame it.
type rendered struct {
	open, body, close string
	// status is missing or unreadable when there is nothing to include.
	status model.AIAttachmentStatus
	// attrs is the attribute text of the tag, reused for self-closing tags.
	attrs string
	// bodyRead is set when the body already stops at the limit it was
	// read with (files).
	cut bool
}

func quoteAttr(s string) string {
	s = strings.NewReplacer(`&`, "&amp;", `"`, "&quot;", `<`, "&lt;", `>`, "&gt;", "\n", " ").Replace(s)
	return s
}

func kindTag(kind model.AIAttachmentKind) string {
	switch kind {
	case model.AIAttachResource, model.AIAttachRepoFile:
		return "file"
	default:
		return string(kind)
	}
}

func selfClosing(kind model.AIAttachmentKind, label, flag string) string {
	return fmt.Sprintf(`<attachment kind="%s" name="%s" %s/>`, kindTag(kind), quoteAttr(label), flag)
}

func cutUTF8(s string, limit int) (string, bool) {
	if len(s) <= limit {
		return s, false
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut], true
}

// build creates the provider messages of a chat and says how every
// attachment was included. msgs are the live messages in order.
func (p *aiPrompt) build(ctx context.Context, session model.Session, msgs []model.AIMessage,
	attachments map[string][]model.AIAttachment) ([]llm.Message, map[string]model.AIAttachmentStatus, error) {
	statuses := map[string]model.AIAttachmentStatus{}
	prefix := make(map[string]string, len(msgs))
	remaining := aiTotalLimit
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		if m.Role != model.AIRoleUser {
			continue
		}
		var b strings.Builder
		for _, a := range attachments[m.ID] {
			text, status, used, err := p.attachmentText(ctx, session, a, remaining)
			if err != nil {
				return nil, nil, err
			}
			remaining -= used
			statuses[a.ID] = status
			b.WriteString(text)
			b.WriteString("\n\n")
		}
		prefix[m.ID] = b.String()
	}
	var out []llm.Message
	for _, m := range msgs {
		switch m.Role {
		case model.AIRoleUser:
			out = append(out, llm.Message{Role: llm.RoleUser, Content: prefix[m.ID] + m.Content})
		case model.AIRoleAssistant:
			if m.Status == model.AIStatusError || m.Status == model.AIStatusStreaming || strings.TrimSpace(m.Content) == "" {
				continue
			}
			out = append(out, llm.Message{Role: llm.RoleAssistant, Content: m.Content})
		}
	}
	return out, statuses, nil
}

// attachmentText renders one attachment within the remaining budget. used is
// the number of body bytes it spent.
func (p *aiPrompt) attachmentText(ctx context.Context, session model.Session, a model.AIAttachment, remaining int) (string, model.AIAttachmentStatus, int, error) {
	if remaining < aiMinBudget {
		return selfClosing(a.Kind, a.Label, `omitted="too much context"`), model.AIAttachOmitted, 0, nil
	}
	limit := min(aiAttachmentLimit, remaining)
	r, err := p.render(ctx, session, a, limit)
	if err != nil {
		return "", "", 0, err
	}
	switch r.status {
	case model.AIAttachMissing:
		return selfClosing(a.Kind, a.Label, `missing="true"`), model.AIAttachMissing, 0, nil
	case model.AIAttachUnreadable:
		return selfClosing(a.Kind, a.Label, `unreadable="true"`), model.AIAttachUnreadable, 0, nil
	}
	body, cut := cutUTF8(r.body, limit)
	cut = cut || r.cut
	status := model.AIAttachIncluded
	if cut {
		body += aiTruncatedMarker
		status = model.AIAttachTruncated
	}
	return r.open + "\n" + body + "\n" + r.close, status, len(body), nil
}

func (p *aiPrompt) render(ctx context.Context, session model.Session, a model.AIAttachment, limit int) (rendered, error) {
	var (
		r   rendered
		err error
	)
	switch a.Kind {
	case model.AIAttachTask:
		r, err = p.renderTask(ctx, session, a.TargetID)
	case model.AIAttachPlan:
		r, err = p.renderPlan(ctx, session, a.TargetID)
	case model.AIAttachProject:
		r, err = p.renderProject(ctx, session, a.TargetID)
	case model.AIAttachResource:
		r, err = p.renderResource(ctx, session, a, limit)
	case model.AIAttachRepoFile:
		r, err = p.renderRepoFile(ctx, session, a, limit)
	default:
		return rendered{}, fmt.Errorf("%w: attachment kind", ErrInvalidInput)
	}
	if errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows) {
		return rendered{status: model.AIAttachMissing}, nil
	}
	return r, err
}

func day(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.UTC().Format("2006-01-02")
}

func taskKey(identifier string, number int) string {
	if identifier == "" || number == 0 {
		return ""
	}
	return fmt.Sprintf("%s-%d", identifier, number)
}

func joinNonEmpty(sep string, parts ...string) string {
	var kept []string
	for _, p := range parts {
		if p != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}

func priorityText(p model.TaskPriority) string {
	if p == "" || p == model.PriorityNone {
		return ""
	}
	return string(p)
}

func firstRunes(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

func (p *aiPrompt) renderTask(ctx context.Context, session model.Session, id string) (rendered, error) {
	ws := session.WorkspaceID
	t, err := p.src.Tasks.Get(ctx, p.db, ws, id)
	if err != nil {
		return rendered{}, err
	}
	project, err := p.src.Projects.Get(ctx, p.db, ws, t.ProjectID)
	if err != nil {
		return rendered{}, err
	}
	plan, err := p.src.Projects.GetPlan(ctx, p.db, ws, t.ProjectID, t.PlanID)
	if err != nil {
		return rendered{}, err
	}
	status, err := p.src.Activity.StatusTitle(ctx, p.db, ws, t.StatusID)
	if err != nil {
		return rendered{}, err
	}
	var lines []string
	lines = append(lines, joinNonEmpty(" · ",
		"Project: "+project.Name, "Plan: "+plan.Name, "Status: "+status,
		ifSet("Priority: ", priorityText(t.Priority))))
	completed := "no"
	if t.CompletedAt != nil {
		completed = "yes"
	}
	lines = append(lines, joinNonEmpty(" · ", ifSet("Start: ", day(t.StartDate)), ifSet("Due: ", day(t.DueDate)), "Completed: "+completed))

	labelIDs, err := p.src.Tasks.LabelIDsByTask(ctx, p.db, ws, []string{t.ID})
	if err != nil {
		return rendered{}, err
	}
	if ids := labelIDs[t.ID]; len(ids) > 0 {
		names, err := p.src.Activity.LabelNames(ctx, p.db, ws, ids)
		if err != nil {
			return rendered{}, err
		}
		var list []string
		for _, lid := range ids {
			if n := names[lid]; n != "" {
				list = append(list, n)
			}
		}
		if len(list) > 0 {
			lines = append(lines, "Labels: "+strings.Join(list, ", "))
		}
	}
	if strings.TrimSpace(t.Description) != "" {
		lines = append(lines, "Description:", strings.TrimSpace(t.Description))
	}
	subs, err := p.src.Tasks.ListSubtasks(ctx, p.db, ws, t.ID)
	if err != nil {
		return rendered{}, err
	}
	if len(subs) > 0 {
		lines = append(lines, "Subtasks:")
		for _, s := range subs {
			box := "[ ]"
			if s.CompletedAt != nil {
				box = "[x]"
			}
			lines = append(lines, joinNonEmpty(" ", "-", box, taskKey(project.Identifier, s.Number), s.Title))
		}
	}
	comments, err := p.src.Comments.ListByTask(ctx, p.db, ws, t.ID)
	if err != nil {
		return rendered{}, err
	}
	if len(comments) > 0 {
		if len(comments) > aiMaxComments {
			comments = comments[len(comments)-aiMaxComments:]
		}
		lines = append(lines, fmt.Sprintf("Comments (latest %d, oldest first):", aiMaxComments))
		for _, c := range comments {
			lines = append(lines, fmt.Sprintf("- %s, %s: %s", c.AuthorName, c.CreatedAt.UTC().Format("2006-01-02"), strings.TrimSpace(c.Body)))
		}
	}
	files, err := p.src.Resources.TaskResources(ctx, session, t.ID)
	if err != nil {
		return rendered{}, err
	}
	if len(files) > 0 {
		var names []string
		for _, f := range files {
			names = append(names, f.Name)
		}
		lines = append(lines, "Linked files: "+strings.Join(names, ", "))
	}
	key := taskKey(project.Identifier, t.Number)
	open := fmt.Sprintf(`<attachment kind="task" key="%s" title="%s">`, quoteAttr(key), quoteAttr(t.Title))
	return rendered{open: open, body: strings.Join(lines, "\n"), close: "</attachment>"}, nil
}

func ifSet(prefix, v string) string {
	if v == "" {
		return ""
	}
	return prefix + v
}

// planLines lists the tasks of a plan grouped by status; subtasks are
// indented.
func (p *aiPrompt) planLines(ctx context.Context, ws string, project model.Project, planID string) ([]string, error) {
	statuses, err := p.src.Projects.ListStatuses(ctx, p.db, ws, planID)
	if err != nil {
		return nil, err
	}
	titles := make(map[string]string, len(statuses))
	for _, s := range statuses {
		titles[s.ID] = s.Title
	}
	tasks, err := p.src.Tasks.ListByPlan(ctx, p.db, ws, planID)
	if err != nil {
		return nil, err
	}
	line := func(indent string, t model.Task, status string) string {
		meta := joinNonEmpty(" · ", status, priorityText(t.Priority), ifSet("due ", day(t.DueDate)))
		text := fmt.Sprintf("%s- %s [%s] %s", indent, taskKey(project.Identifier, t.Number), meta, t.Title)
		text = strings.Replace(text, "-  [", "- [", 1)
		if d := firstRunes(t.Description, aiPlanLineDescRune); d != "" {
			text += " — " + d
		}
		return text
	}
	var lines []string
	for _, t := range tasks {
		lines = append(lines, line("", t, titles[t.StatusID]))
		if t.SubtaskCount == 0 {
			continue
		}
		subs, err := p.src.Tasks.ListSubtasks(ctx, p.db, ws, t.ID)
		if err != nil {
			return nil, err
		}
		for _, s := range subs {
			lines = append(lines, line("  ", s, titles[s.StatusID]))
		}
	}
	return lines, nil
}

func (p *aiPrompt) renderPlan(ctx context.Context, session model.Session, id string) (rendered, error) {
	ws := session.WorkspaceID
	projects, err := p.src.Placement.LivePlanProjects(ctx, p.db, ws, []string{id})
	if err != nil {
		return rendered{}, err
	}
	projectID, ok := projects[id]
	if !ok {
		return rendered{}, ErrNotFound
	}
	project, err := p.src.Projects.Get(ctx, p.db, ws, projectID)
	if err != nil {
		return rendered{}, err
	}
	plan, err := p.src.Projects.GetPlan(ctx, p.db, ws, projectID, id)
	if err != nil {
		return rendered{}, err
	}
	lines, err := p.planLines(ctx, ws, project, id)
	if err != nil {
		return rendered{}, err
	}
	open := fmt.Sprintf(`<attachment kind="plan" project="%s" name="%s">`, quoteAttr(project.Name), quoteAttr(plan.Name))
	return rendered{open: open, body: strings.Join(lines, "\n"), close: "</attachment>"}, nil
}

func (p *aiPrompt) renderProject(ctx context.Context, session model.Session, id string) (rendered, error) {
	ws := session.WorkspaceID
	project, err := p.src.Projects.Get(ctx, p.db, ws, id)
	if err != nil {
		return rendered{}, err
	}
	plans, err := p.src.Projects.ListPlans(ctx, p.db, ws, id)
	if err != nil {
		return rendered{}, err
	}
	var lines []string
	for _, pl := range plans {
		lines = append(lines, "## Plan: "+pl.Name)
		body, err := p.planLines(ctx, ws, project, pl.ID)
		if err != nil {
			return rendered{}, err
		}
		lines = append(lines, body...)
	}
	open := fmt.Sprintf(`<attachment kind="project" name="%s">`, quoteAttr(project.Name))
	return rendered{open: open, body: strings.Join(lines, "\n"), close: "</attachment>"}, nil
}

// readText reads up to limit bytes of a file as text; cut says there was
// more.
func readText(f *os.File, limit int) (string, bool, error) {
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, int64(limit)+1))
	if err != nil {
		return "", false, err
	}
	text := strings.ToValidUTF8(string(data), "�")
	text, cut := cutUTF8(text, limit)
	return text, cut, nil
}

func (p *aiPrompt) renderResource(ctx context.Context, session model.Session, a model.AIAttachment, limit int) (rendered, error) {
	res, err := p.src.Resources.Get(ctx, session, a.TargetID)
	if err != nil {
		return rendered{}, err
	}
	if res.TrashedAt != nil {
		return rendered{}, ErrNotFound
	}
	if res.Kind != model.ResourceFile || !res.IsText {
		return rendered{status: model.AIAttachUnreadable}, nil
	}
	_, f, err := p.src.Resources.Open(ctx, session, a.TargetID)
	if errors.Is(err, ErrFileMissing) {
		return rendered{status: model.AIAttachUnreadable}, nil
	}
	if err != nil {
		return rendered{}, err
	}
	text, cut, err := readText(f, limit)
	if err != nil {
		return rendered{status: model.AIAttachUnreadable}, nil
	}
	open := fmt.Sprintf(`<attachment kind="file" name="%s">`, quoteAttr(res.Name))
	return rendered{open: open, body: text, close: "</attachment>", cut: cut}, nil
}

func (p *aiPrompt) renderRepoFile(ctx context.Context, session model.Session, a model.AIAttachment, limit int) (rendered, error) {
	repo, err := p.src.Repos.Get(ctx, session, a.TargetID)
	if err != nil {
		return rendered{}, err
	}
	if repo.TrashedAt != nil {
		return rendered{}, ErrNotFound
	}
	entry, f, err := p.src.Repos.OpenFile(ctx, session, a.TargetID, a.Path)
	if errors.Is(err, ErrNotFound) {
		return rendered{}, err
	}
	if err != nil {
		return rendered{status: model.AIAttachUnreadable}, nil
	}
	if !entry.IsText {
		f.Close()
		return rendered{status: model.AIAttachUnreadable}, nil
	}
	text, cut, err := readText(f, limit)
	if err != nil {
		return rendered{status: model.AIAttachUnreadable}, nil
	}
	open := fmt.Sprintf(`<attachment kind="file" repository="%s" path="%s">`, quoteAttr(repo.FullName), quoteAttr(a.Path))
	return rendered{open: open, body: text, close: "</attachment>", cut: cut}, nil
}

// target checks that an attachment target exists and can be attached, and
// returns its display label.
func (p *aiPrompt) target(ctx context.Context, session model.Session, kind model.AIAttachmentKind, targetID, filePath string) (string, error) {
	ws := session.WorkspaceID
	if targetID == "" {
		return "", fmt.Errorf("%w: attachment target", ErrInvalidInput)
	}
	if kind != model.AIAttachRepoFile && filePath != "" {
		return "", fmt.Errorf("%w: attachment path", ErrInvalidInput)
	}
	switch kind {
	case model.AIAttachTask:
		t, err := p.src.Tasks.Get(ctx, p.db, ws, targetID)
		if err != nil {
			return "", notFound(err, "task")
		}
		project, err := p.src.Projects.Get(ctx, p.db, ws, t.ProjectID)
		if err != nil {
			return "", notFound(err, "task")
		}
		return joinNonEmpty(" ", taskKey(project.Identifier, t.Number), t.Title), nil
	case model.AIAttachPlan:
		projects, err := p.src.Placement.LivePlanProjects(ctx, p.db, ws, []string{targetID})
		if err != nil {
			return "", err
		}
		projectID, ok := projects[targetID]
		if !ok {
			return "", fmt.Errorf("%w: plan", ErrNotFound)
		}
		project, err := p.src.Projects.Get(ctx, p.db, ws, projectID)
		if err != nil {
			return "", notFound(err, "plan")
		}
		plan, err := p.src.Projects.GetPlan(ctx, p.db, ws, projectID, targetID)
		if err != nil {
			return "", notFound(err, "plan")
		}
		return project.Name + " / " + plan.Name, nil
	case model.AIAttachProject:
		project, err := p.src.Projects.Get(ctx, p.db, ws, targetID)
		if err != nil {
			return "", notFound(err, "project")
		}
		return project.Name, nil
	case model.AIAttachResource:
		res, err := p.src.Resources.Get(ctx, session, targetID)
		if err != nil {
			return "", err
		}
		if res.TrashedAt != nil {
			return "", fmt.Errorf("%w: resource", ErrNotFound)
		}
		if res.Kind != model.ResourceFile || !res.IsText {
			return "", fmt.Errorf("%w: resource is not a text file", ErrInvalidInput)
		}
		return res.Name, nil
	case model.AIAttachRepoFile:
		if filePath == "" {
			return "", fmt.Errorf("%w: attachment path", ErrInvalidInput)
		}
		repo, err := p.src.Repos.Get(ctx, session, targetID)
		if err != nil {
			return "", err
		}
		if repo.TrashedAt != nil {
			return "", fmt.Errorf("%w: repository", ErrNotFound)
		}
		entry, f, err := p.src.Repos.OpenFile(ctx, session, targetID, filePath)
		if err != nil {
			if errors.Is(err, ErrFileMissing) {
				return "", fmt.Errorf("%w: repository is not downloaded yet", ErrInvalidInput)
			}
			return "", err
		}
		f.Close()
		if !entry.IsText {
			return "", fmt.Errorf("%w: file is not text", ErrInvalidInput)
		}
		return repo.FullName + "/" + entry.Path, nil
	default:
		return "", fmt.Errorf("%w: attachment kind", ErrInvalidInput)
	}
}
