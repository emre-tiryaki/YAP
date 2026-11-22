package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/emre-tiryaki/YAP/backend/internal/database"
	"github.com/emre-tiryaki/YAP/backend/internal/llm"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

const (
	maxAIMessageRunes  = 32_000
	maxAIAttachments   = 20
	maxAIModelLength   = 200
	maxAIChatTitle     = 200
	aiTitleRunes       = 60
	maxAIChatSearch    = 200
	aiFlushInterval    = time.Second
	aiFinishTimeout    = 5 * time.Second
	aiStopWait         = 5 * time.Second
	aiErrorCodeDefault = "internal"
)

type AIRepository interface {
	ListChats(ctx context.Context, q database.DBTX, workspaceID, userID, needle string) ([]model.AIChat, error)
	GetChat(ctx context.Context, q database.DBTX, workspaceID, userID, id string) (model.AIChat, error)
	InsertChat(ctx context.Context, q database.DBTX, c model.AIChat, now string) error
	UpdateChatTitle(ctx context.Context, q database.DBTX, workspaceID, id, title, now string) error
	UpdateChatModel(ctx context.Context, q database.DBTX, workspaceID, id, connectionID, modelName, now string) error
	TouchChat(ctx context.Context, q database.DBTX, workspaceID, id, now string) error
	SoftDeleteChat(ctx context.Context, q database.DBTX, workspaceID, id, now string) error

	ListMessages(ctx context.Context, q database.DBTX, workspaceID, chatID string) ([]model.AIMessage, error)
	InsertMessage(ctx context.Context, q database.DBTX, m model.AIMessage, now string) error
	UpdateMessageContent(ctx context.Context, q database.DBTX, workspaceID, id, content, now string) error
	FinishMessage(ctx context.Context, q database.DBTX, workspaceID, id, content string, status model.AIMessageStatus, errorCode, now string) error
	SoftDeleteMessage(ctx context.Context, q database.DBTX, workspaceID, id, now string) error
	MarkStreamingInterrupted(ctx context.Context, q database.DBTX, now string) (int64, error)

	InsertAttachment(ctx context.Context, q database.DBTX, a model.AIAttachment, now string) error
	ListAttachments(ctx context.Context, q database.DBTX, workspaceID string, messageIDs []string) (map[string][]model.AIAttachment, error)
	SetAttachmentStatuses(ctx context.Context, q database.DBTX, workspaceID string, statuses map[string]model.AIAttachmentStatus, now string) error

	LiveTaskPlacements(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]model.AITaskPlacement, error)
	LivePlanProjects(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]string, error)
	LiveProjectIDs(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]bool, error)
	LiveResourceIDs(ctx context.Context, q database.DBTX, workspaceID string, ids []string) (map[string]bool, error)
}

type AIAttachmentInput struct {
	Kind     model.AIAttachmentKind
	TargetID string
	Path     string
}

type SendAIInput struct {
	ConnectionID string
	Model        string
	Content      string
	Attachments  []AIAttachmentInput
}

type RegenerateAIInput struct {
	ConnectionID string
	Model        string
}

type AISendResult struct {
	Chat             model.AIChat
	UserMessage      model.AIMessage
	AssistantMessage model.AIMessage
}

// AIAttachmentView is an attachment with what the page needs to link it.
type AIAttachmentView struct {
	model.AIAttachment
	// Missing is true when the target is gone or in the trash.
	Missing   bool
	ProjectID string
	PlanID    string
}

type AIMessageView struct {
	model.AIMessage
	Attachments []AIAttachmentView
}

type AIChatDetail struct {
	Chat     model.AIChat
	Messages []AIMessageView
}

// generation is the answer being written right now.
type generation struct {
	chatID string

	mu        sync.Mutex
	messageID string
	cancel    context.CancelFunc
	content   strings.Builder
	changed   chan struct{}
	done      bool
	status    model.AIMessageStatus
	stopped   bool

	finished     chan struct{}
	finishedOnce sync.Once
}

func newGeneration(chatID string) *generation {
	return &generation{chatID: chatID, changed: make(chan struct{}), finished: make(chan struct{})}
}

func (g *generation) append(text string) {
	g.mu.Lock()
	g.content.WriteString(text)
	close(g.changed)
	g.changed = make(chan struct{})
	g.mu.Unlock()
}

func (g *generation) text() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.content.String()
}

func (g *generation) markStopped() {
	g.mu.Lock()
	g.stopped = true
	cancel := g.cancel
	g.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (g *generation) wasStopped() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.stopped
}

func (g *generation) finish(status model.AIMessageStatus) {
	g.mu.Lock()
	g.done = true
	g.status = status
	close(g.changed)
	g.mu.Unlock()
	g.finishedOnce.Do(func() { close(g.finished) })
}

// AIStream is a view of the generation of one chat for the stream endpoint.
type AIStream struct{ g *generation }

func (s AIStream) MessageID() string {
	s.g.mu.Lock()
	defer s.g.mu.Unlock()
	return s.g.messageID
}

// Read returns the text written after offset, the new offset, a channel
// that closes on the next change, and whether the answer is finished.
func (s AIStream) Read(offset int) (text string, next int, changed <-chan struct{}, done bool, status model.AIMessageStatus) {
	g := s.g
	g.mu.Lock()
	defer g.mu.Unlock()
	all := g.content.String()
	if offset > len(all) {
		offset = len(all)
	}
	return all[offset:], len(all), g.changed, g.done, g.status
}

type AIChatService struct {
	db       *sql.DB
	repo     AIRepository
	conns    *AIConnectionService
	prompt   *aiPrompt
	settings SettingsRepository
	llm      LLM

	mu         sync.Mutex
	active     *generation
	background context.Context
	wg         sync.WaitGroup
}

func NewAIChatService(db *sql.DB, repo AIRepository, conns *AIConnectionService, prompt *aiPrompt,
	settings SettingsRepository, l LLM) *AIChatService {
	return &AIChatService{db: db, repo: repo, conns: conns, prompt: prompt, settings: settings, llm: l,
		background: context.Background()}
}

// NewAIPrompt builds the attachment reader the chat service needs.
func NewAIPrompt(db *sql.DB, src AIContextSources) *aiPrompt {
	return newAIPrompt(db, src)
}

// Start sets the context generations run in; cancelling it interrupts them.
func (s *AIChatService) Start(ctx context.Context) {
	s.mu.Lock()
	s.background = ctx
	s.mu.Unlock()
}

// Wait blocks until running generations have written their final state.
func (s *AIChatService) Wait() { s.wg.Wait() }

// RecoverInterrupted marks answers a restart left half-written.
func (s *AIChatService) RecoverInterrupted(ctx context.Context) error {
	_, err := s.repo.MarkStreamingInterrupted(ctx, s.db, nowUTC())
	return err
}

// Status returns the chat that is being answered, or "".
func (s *AIChatService) Status() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == nil {
		return ""
	}
	return s.active.chatID
}

// StopAll stops the running generation of any chat (used before a reset).
func (s *AIChatService) StopAll() {
	s.mu.Lock()
	g := s.active
	s.mu.Unlock()
	if g == nil {
		return
	}
	g.markStopped()
	select {
	case <-g.finished:
	case <-time.After(aiStopWait):
	}
}

// Subscribe returns the running generation of a chat.
func (s *AIChatService) Subscribe(chatID string) (AIStream, bool) {
	s.mu.Lock()
	g := s.active
	s.mu.Unlock()
	if g == nil || g.chatID != chatID {
		return AIStream{}, false
	}
	st := AIStream{g}
	if st.MessageID() == "" {
		return AIStream{}, false
	}
	return st, true
}

func (s *AIChatService) reserve(chatID string) (*generation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active != nil {
		return nil, ErrAIBusy
	}
	g := newGeneration(chatID)
	s.active = g
	return g, nil
}

// release gives the reservation back when no goroutine took it over.
func (s *AIChatService) release(g *generation) {
	s.mu.Lock()
	if s.active == g {
		s.active = nil
	}
	s.mu.Unlock()
	g.finishedOnce.Do(func() { close(g.finished) })
}

func (s *AIChatService) getChat(ctx context.Context, q database.DBTX, session model.Session, id string) (model.AIChat, error) {
	c, err := s.repo.GetChat(ctx, q, session.WorkspaceID, session.UserID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AIChat{}, fmt.Errorf("%w: chat", ErrNotFound)
	}
	return c, err
}

func (s *AIChatService) List(ctx context.Context, session model.Session, q string) ([]model.AIChat, error) {
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > maxAIChatSearch {
		return nil, fmt.Errorf("%w: search is too long", ErrInvalidInput)
	}
	chats, err := s.repo.ListChats(ctx, s.db, session.WorkspaceID, session.UserID, database.Fold(q))
	if err != nil {
		return nil, err
	}
	active := s.Status()
	for i := range chats {
		chats[i].Generating = active != "" && chats[i].ID == active
	}
	return chats, nil
}

func (s *AIChatService) Get(ctx context.Context, session model.Session, chatID string) (AIChatDetail, error) {
	ws := session.WorkspaceID
	chat, err := s.getChat(ctx, s.db, session, chatID)
	if err != nil {
		return AIChatDetail{}, err
	}
	chat.Generating = s.Status() == chat.ID
	msgs, err := s.repo.ListMessages(ctx, s.db, ws, chat.ID)
	if err != nil {
		return AIChatDetail{}, err
	}
	ids := make([]string, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	atts, err := s.repo.ListAttachments(ctx, s.db, ws, ids)
	if err != nil {
		return AIChatDetail{}, err
	}
	var tasks, plans, projects, resources []string
	for _, list := range atts {
		for _, a := range list {
			switch a.Kind {
			case model.AIAttachTask:
				tasks = append(tasks, a.TargetID)
			case model.AIAttachPlan:
				plans = append(plans, a.TargetID)
			case model.AIAttachProject:
				projects = append(projects, a.TargetID)
			default:
				resources = append(resources, a.TargetID)
			}
		}
	}
	taskLive, err := s.repo.LiveTaskPlacements(ctx, s.db, ws, tasks)
	if err != nil {
		return AIChatDetail{}, err
	}
	planLive, err := s.repo.LivePlanProjects(ctx, s.db, ws, plans)
	if err != nil {
		return AIChatDetail{}, err
	}
	projectLive, err := s.repo.LiveProjectIDs(ctx, s.db, ws, projects)
	if err != nil {
		return AIChatDetail{}, err
	}
	resourceLive, err := s.repo.LiveResourceIDs(ctx, s.db, ws, resources)
	if err != nil {
		return AIChatDetail{}, err
	}
	out := AIChatDetail{Chat: chat, Messages: make([]AIMessageView, 0, len(msgs))}
	for _, m := range msgs {
		view := AIMessageView{AIMessage: m, Attachments: []AIAttachmentView{}}
		for _, a := range atts[m.ID] {
			v := AIAttachmentView{AIAttachment: a}
			switch a.Kind {
			case model.AIAttachTask:
				place, ok := taskLive[a.TargetID]
				v.Missing, v.ProjectID, v.PlanID = !ok, place.ProjectID, place.PlanID
			case model.AIAttachPlan:
				projectID, ok := planLive[a.TargetID]
				v.Missing, v.ProjectID, v.PlanID = !ok, projectID, a.TargetID
			case model.AIAttachProject:
				v.Missing, v.ProjectID = !projectLive[a.TargetID], a.TargetID
			default:
				v.Missing = !resourceLive[a.TargetID]
			}
			view.Attachments = append(view.Attachments, v)
		}
		out.Messages = append(out.Messages, view)
	}
	return out, nil
}

func chatTitle(content string) string {
	line := ""
	for _, l := range strings.Split(content, "\n") {
		if l = strings.Join(strings.Fields(l), " "); l != "" {
			line = l
			break
		}
	}
	if utf8.RuneCountInString(line) <= aiTitleRunes {
		return line
	}
	return string([]rune(line)[:aiTitleRunes]) + "…"
}

func (in SendAIInput) validate() (SendAIInput, error) {
	in.Content = strings.TrimSpace(in.Content)
	if n := utf8.RuneCountInString(in.Content); n == 0 || n > maxAIMessageRunes {
		return in, fmt.Errorf("%w: message", ErrInvalidInput)
	}
	if len(in.Attachments) > maxAIAttachments {
		return in, fmt.Errorf("%w: too many attachments", ErrInvalidInput)
	}
	return in, validateAIModel(in.ConnectionID, in.Model)
}

func validateAIModel(connectionID, modelName string) error {
	if connectionID == "" || modelName == "" || utf8.RuneCountInString(modelName) > maxAIModelLength {
		return fmt.Errorf("%w: model", ErrInvalidInput)
	}
	return nil
}

// Create starts a chat with its first message.
func (s *AIChatService) Create(ctx context.Context, session model.Session, in SendAIInput) (AISendResult, error) {
	in, err := in.validate()
	if err != nil {
		return AISendResult{}, err
	}
	return s.send(ctx, session, "", in)
}

// Send adds a message to a chat and starts the answer.
func (s *AIChatService) Send(ctx context.Context, session model.Session, chatID string, in SendAIInput) (AISendResult, error) {
	in, err := in.validate()
	if err != nil {
		return AISendResult{}, err
	}
	return s.send(ctx, session, chatID, in)
}

func (s *AIChatService) send(ctx context.Context, session model.Session, chatID string, in SendAIInput) (res AISendResult, err error) {
	ws := session.WorkspaceID
	creating := chatID == ""
	if creating {
		chatID = model.NewID()
	}
	g, err := s.reserve(chatID)
	if err != nil {
		return AISendResult{}, err
	}
	started := false
	defer func() {
		if !started {
			s.release(g)
		}
	}()

	var chat model.AIChat
	if !creating {
		if chat, err = s.getChat(ctx, s.db, session, chatID); err != nil {
			return AISendResult{}, err
		}
	}
	_, ep, err := s.conns.endpoint(ctx, session, in.ConnectionID)
	if err != nil {
		return AISendResult{}, err
	}

	labels := make([]string, len(in.Attachments))
	for i, a := range in.Attachments {
		if labels[i], err = s.prompt.target(ctx, session, a.Kind, a.TargetID, a.Path); err != nil {
			return AISendResult{}, err
		}
	}

	now := nowUTC()
	user := model.AIMessage{ID: model.NewID(), WorkspaceID: ws, ChatID: chatID, Role: model.AIRoleUser, Content: in.Content, Status: model.AIStatusComplete}
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if creating {
			chat = model.AIChat{ID: chatID, WorkspaceID: ws, UserID: session.UserID, Title: chatTitle(in.Content),
				ConnectionID: in.ConnectionID, Model: in.Model}
			if err := s.repo.InsertChat(ctx, tx, chat, now); err != nil {
				return err
			}
		} else if err := s.repo.UpdateChatModel(ctx, tx, ws, chatID, in.ConnectionID, in.Model, now); err != nil {
			return err
		}
		if err := s.repo.InsertMessage(ctx, tx, user, now); err != nil {
			return err
		}
		for i, a := range in.Attachments {
			att := model.AIAttachment{ID: model.NewID(), WorkspaceID: ws, MessageID: user.ID, Position: i,
				Kind: a.Kind, TargetID: a.TargetID, Path: a.Path, Label: labels[i]}
			if err := s.repo.InsertAttachment(ctx, tx, att, now); err != nil {
				return err
			}
		}
		return s.repo.TouchChat(ctx, tx, ws, chatID, now)
	})
	if err != nil {
		return AISendResult{}, err
	}

	assistant, err := s.begin(ctx, session, g, chatID, in.ConnectionID, in.Model, ep)
	if err != nil {
		return AISendResult{}, err
	}
	started = true
	chat, err = s.getChat(ctx, s.db, session, chatID)
	if err != nil {
		return AISendResult{}, err
	}
	chat.Generating = true
	user.CreatedAt, user.UpdatedAt = parseStamp(now), parseStamp(now)
	return AISendResult{Chat: chat, UserMessage: user, AssistantMessage: assistant}, nil
}

// begin builds the prompt from the stored chat, stores the empty answer
// and starts the goroutine that fills it.
func (s *AIChatService) begin(ctx context.Context, session model.Session, g *generation, chatID, connectionID, modelName string, ep llm.Endpoint) (model.AIMessage, error) {
	ws := session.WorkspaceID
	msgs, err := s.repo.ListMessages(ctx, s.db, ws, chatID)
	if err != nil {
		return model.AIMessage{}, err
	}
	ids := make([]string, len(msgs))
	for i, m := range msgs {
		ids[i] = m.ID
	}
	atts, err := s.repo.ListAttachments(ctx, s.db, ws, ids)
	if err != nil {
		return model.AIMessage{}, err
	}
	settings, err := loadUserSettings(ctx, s.db, s.settings, session.UserID)
	if err != nil {
		return model.AIMessage{}, err
	}
	messages, statuses, err := s.prompt.build(ctx, session, msgs, atts)
	if err != nil {
		return model.AIMessage{}, err
	}
	now := nowUTC()
	assistant := model.AIMessage{ID: model.NewID(), WorkspaceID: ws, ChatID: chatID, Role: model.AIRoleAssistant,
		Status: model.AIStatusStreaming, ConnectionID: connectionID, Model: modelName}
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := s.repo.InsertMessage(ctx, tx, assistant, now); err != nil {
			return err
		}
		return s.repo.SetAttachmentStatuses(ctx, tx, ws, statuses, now)
	})
	if err != nil {
		return model.AIMessage{}, err
	}
	assistant.CreatedAt, assistant.UpdatedAt = parseStamp(now), parseStamp(now)

	s.mu.Lock()
	base := s.background
	s.mu.Unlock()
	runCtx, cancel := context.WithCancel(base)
	g.mu.Lock()
	g.messageID = assistant.ID
	g.cancel = cancel
	stopped := g.stopped
	g.mu.Unlock()
	if stopped {
		cancel()
	}
	req := llm.Request{Endpoint: ep, Model: modelName, System: aiSystem(settings.AIInstructions), Messages: messages}
	s.wg.Add(1)
	go s.run(runCtx, cancel, session, g, assistant.ID, req)
	return assistant, nil
}

func aiErrorCode(err error) string {
	switch {
	case errors.Is(err, model.ErrAIUnreachable):
		return "ai_unreachable"
	case errors.Is(err, model.ErrAIAuthFailed):
		return "ai_auth_failed"
	case errors.Is(err, model.ErrAIRateLimited):
		return "ai_rate_limited"
	case errors.Is(err, model.ErrAIModelNotFound):
		return "ai_model_not_found"
	case errors.Is(err, model.ErrAIRejected):
		return "ai_rejected"
	default:
		return aiErrorCodeDefault
	}
}

func (s *AIChatService) run(ctx context.Context, cancel context.CancelFunc, session model.Session, g *generation, messageID string, req llm.Request) {
	defer s.wg.Done()
	defer cancel()
	ws := session.WorkspaceID

	stopFlush, flushDone := make(chan struct{}), make(chan struct{})
	go func() {
		defer close(flushDone)
		t := time.NewTicker(aiFlushInterval)
		defer t.Stop()
		written := 0
		for {
			select {
			case <-t.C:
				content := g.text()
				if len(content) == written {
					continue
				}
				wctx, wcancel := context.WithTimeout(context.WithoutCancel(ctx), aiFinishTimeout)
				if err := s.repo.UpdateMessageContent(wctx, s.db, ws, messageID, content, nowUTC()); err != nil {
					slog.Error("flush ai message", "error", err)
				} else {
					written = len(content)
				}
				wcancel()
			case <-stopFlush:
				return
			}
		}
	}()

	var streamErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("ai generation panicked", "panic", r)
				streamErr = fmt.Errorf("panic: %v", r)
			}
		}()
		streamErr = s.llm.Stream(ctx, req, g.append)
	}()
	close(stopFlush)
	<-flushDone

	content := g.text()
	status, code := model.AIStatusComplete, ""
	switch {
	case streamErr == nil && strings.TrimSpace(content) == "":
		status, code = model.AIStatusError, "ai_rejected"
	case streamErr == nil:
	case ctx.Err() != nil || errors.Is(streamErr, context.Canceled):
		status = model.AIStatusInterrupted
		if g.wasStopped() {
			status = model.AIStatusStopped
		}
	default:
		status, code = model.AIStatusError, aiErrorCode(streamErr)
		if code == aiErrorCodeDefault {
			slog.Error("ai generation failed", "error", streamErr)
		} else {
			slog.Warn("ai generation failed", "code", code, "error", streamErr)
		}
	}

	fctx, fcancel := context.WithTimeout(context.WithoutCancel(ctx), aiFinishTimeout)
	defer fcancel()
	now := nowUTC()
	err := database.WithTx(fctx, s.db, func(tx *sql.Tx) error {
		if err := s.repo.FinishMessage(fctx, tx, ws, messageID, content, status, code, now); err != nil {
			return err
		}
		return s.repo.TouchChat(fctx, tx, ws, g.chatID, now)
	})
	if err != nil {
		slog.Error("finish ai message", "error", err)
	}
	s.mu.Lock()
	if s.active == g {
		s.active = nil
	}
	s.mu.Unlock()
	g.finish(status)
}

// Regenerate answers the last user message again, replacing the last answer.
func (s *AIChatService) Regenerate(ctx context.Context, session model.Session, chatID string, in RegenerateAIInput) (model.AIMessage, error) {
	if err := validateAIModel(in.ConnectionID, in.Model); err != nil {
		return model.AIMessage{}, err
	}
	ws := session.WorkspaceID
	g, err := s.reserve(chatID)
	if err != nil {
		return model.AIMessage{}, err
	}
	started := false
	defer func() {
		if !started {
			s.release(g)
		}
	}()
	if _, err := s.getChat(ctx, s.db, session, chatID); err != nil {
		return model.AIMessage{}, err
	}
	msgs, err := s.repo.ListMessages(ctx, s.db, ws, chatID)
	if err != nil {
		return model.AIMessage{}, err
	}
	if len(msgs) == 0 {
		return model.AIMessage{}, fmt.Errorf("%w: nothing to regenerate", ErrInvalidInput)
	}
	last := msgs[len(msgs)-1]
	if last.Status == model.AIStatusStreaming {
		return model.AIMessage{}, fmt.Errorf("%w: answer is still being written", ErrInvalidInput)
	}
	if last.Role == model.AIRoleAssistant && (len(msgs) < 2 || msgs[len(msgs)-2].Role != model.AIRoleUser) {
		return model.AIMessage{}, fmt.Errorf("%w: nothing to regenerate", ErrInvalidInput)
	}
	_, ep, err := s.conns.endpoint(ctx, session, in.ConnectionID)
	if err != nil {
		return model.AIMessage{}, err
	}
	now := nowUTC()
	err = database.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if last.Role == model.AIRoleAssistant {
			if err := s.repo.SoftDeleteMessage(ctx, tx, ws, last.ID, now); err != nil {
				return err
			}
		}
		if err := s.repo.UpdateChatModel(ctx, tx, ws, chatID, in.ConnectionID, in.Model, now); err != nil {
			return err
		}
		return s.repo.TouchChat(ctx, tx, ws, chatID, now)
	})
	if err != nil {
		return model.AIMessage{}, err
	}
	assistant, err := s.begin(ctx, session, g, chatID, in.ConnectionID, in.Model, ep)
	if err != nil {
		return model.AIMessage{}, err
	}
	started = true
	return assistant, nil
}

// Stop ends the answer being written for a chat now and keeps what was
// written. It does nothing when the chat is not being answered.
func (s *AIChatService) Stop(ctx context.Context, session model.Session, chatID string) error {
	if _, err := s.getChat(ctx, s.db, session, chatID); err != nil {
		return err
	}
	s.stop(chatID)
	return nil
}

func (s *AIChatService) stop(chatID string) {
	s.mu.Lock()
	g := s.active
	s.mu.Unlock()
	if g == nil || g.chatID != chatID {
		return
	}
	g.markStopped()
	select {
	case <-g.finished:
	case <-time.After(aiStopWait):
	}
}

func (s *AIChatService) Rename(ctx context.Context, session model.Session, chatID, title string) (model.AIChat, error) {
	title = strings.TrimSpace(title)
	if n := utf8.RuneCountInString(title); n == 0 || n > maxAIChatTitle {
		return model.AIChat{}, fmt.Errorf("%w: title", ErrInvalidInput)
	}
	chat, err := s.getChat(ctx, s.db, session, chatID)
	if err != nil {
		return model.AIChat{}, err
	}
	if err := s.repo.UpdateChatTitle(ctx, s.db, session.WorkspaceID, chat.ID, title, nowUTC()); err != nil {
		return model.AIChat{}, err
	}
	chat, err = s.getChat(ctx, s.db, session, chatID)
	chat.Generating = s.Status() == chat.ID
	return chat, err
}

func (s *AIChatService) Delete(ctx context.Context, session model.Session, chatID string) error {
	chat, err := s.getChat(ctx, s.db, session, chatID)
	if err != nil {
		return err
	}
	s.stop(chat.ID)
	return s.repo.SoftDeleteChat(ctx, s.db, session.WorkspaceID, chat.ID, nowUTC())
}
