package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

const sseKeepAlive = 15 * time.Second

type AIHandler struct {
	session model.Session
	chats   *service.AIChatService
	conns   *service.AIConnectionService
}

func NewAIHandler(session model.Session, chats *service.AIChatService, conns *service.AIConnectionService) *AIHandler {
	return &AIHandler{session: session, chats: chats, conns: conns}
}

func (h *AIHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/ai/connections", h.listConnections)
	mux.HandleFunc("POST /api/ai/connections", h.createConnection)
	mux.HandleFunc("POST /api/ai/connections/test", h.testConnection)
	mux.HandleFunc("PATCH /api/ai/connections/{id}", h.updateConnection)
	mux.HandleFunc("DELETE /api/ai/connections/{id}", h.deleteConnection)
	mux.HandleFunc("GET /api/ai/connections/{id}/models", h.models)

	mux.HandleFunc("GET /api/ai/status", h.status)
	mux.HandleFunc("GET /api/ai/chats", h.listChats)
	mux.HandleFunc("POST /api/ai/chats", h.createChat)
	mux.HandleFunc("GET /api/ai/chats/{id}", h.getChat)
	mux.HandleFunc("PATCH /api/ai/chats/{id}", h.renameChat)
	mux.HandleFunc("DELETE /api/ai/chats/{id}", h.deleteChat)
	mux.HandleFunc("POST /api/ai/chats/{id}/messages", h.send)
	mux.HandleFunc("POST /api/ai/chats/{id}/regenerate", h.regenerate)
	mux.HandleFunc("POST /api/ai/chats/{id}/stop", h.stop)
	mux.HandleFunc("GET /api/ai/chats/{id}/stream", h.stream)
}

// requireJSON guards the endpoints that make outbound requests: a page on
// another site cannot send JSON to localhost without a CORS preflight, which
// this server does not answer.
func requireJSON(w http.ResponseWriter, r *http.Request) bool {
	if mt := r.Header.Get("Content-Type"); !strings.HasPrefix(mt, "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, errorResponse{Error: "invalid_input"})
		return false
	}
	return true
}

type aiConnectionResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Preset    string `json:"preset"`
	BaseURL   string `json:"base_url"`
	HasKey    bool   `json:"has_key"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func toAIConnectionResponse(c model.AIConnection) aiConnectionResponse {
	return aiConnectionResponse{
		ID: c.ID, Name: c.Name, Provider: string(c.Provider), Preset: string(c.Preset), BaseURL: c.BaseURL, HasKey: c.HasKey,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type modelsResponse struct {
	Models []string `json:"models"`
}

func (h *AIHandler) listConnections(w http.ResponseWriter, r *http.Request) {
	list, err := h.conns.List(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]aiConnectionResponse, 0, len(list))
	for _, c := range list {
		res = append(res, toAIConnectionResponse(c))
	}
	writeJSON(w, http.StatusOK, res)
}

type createAIConnectionRequest struct {
	Name    string `json:"name"`
	Preset  string `json:"preset"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

func (h *AIHandler) createConnection(w http.ResponseWriter, r *http.Request) {
	var req createAIConnectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := h.conns.Create(r.Context(), h.session, service.CreateAIConnectionInput{
		Name: req.Name, Preset: model.AIPreset(req.Preset), BaseURL: req.BaseURL, APIKey: req.APIKey,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toAIConnectionResponse(c))
}

type updateAIConnectionRequest struct {
	Name    *string `json:"name"`
	BaseURL *string `json:"base_url"`
	APIKey  *string `json:"api_key"`
}

func (h *AIHandler) updateConnection(w http.ResponseWriter, r *http.Request) {
	var req updateAIConnectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := h.conns.Update(r.Context(), h.session, r.PathValue("id"), service.UpdateAIConnectionInput{
		Name: req.Name, BaseURL: req.BaseURL, APIKey: req.APIKey,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAIConnectionResponse(c))
}

func (h *AIHandler) deleteConnection(w http.ResponseWriter, r *http.Request) {
	if err := h.conns.Delete(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AIHandler) models(w http.ResponseWriter, r *http.Request) {
	models, err := h.conns.Models(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, modelsResponse{Models: models})
}

type testAIConnectionRequest struct {
	Preset  string `json:"preset"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

func (h *AIHandler) testConnection(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	var req testAIConnectionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	models, err := h.conns.TestUnsaved(r.Context(), service.TestAIConnectionInput{
		Preset: model.AIPreset(req.Preset), BaseURL: req.BaseURL, APIKey: req.APIKey,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, modelsResponse{Models: models})
}

type aiChatResponse struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	ConnectionID *string `json:"connection_id"`
	Model        string  `json:"model"`
	Generating   bool    `json:"generating"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func toAIChatResponse(c model.AIChat) aiChatResponse {
	return aiChatResponse{
		ID: c.ID, Title: c.Title, ConnectionID: optionalString(c.ConnectionID), Model: c.Model, Generating: c.Generating,
		CreatedAt: c.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: c.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type aiAttachmentResponse struct {
	ID         string  `json:"id"`
	Kind       string  `json:"kind"`
	TargetID   string  `json:"target_id"`
	Path       *string `json:"path"`
	Label      string  `json:"label"`
	LastStatus string  `json:"last_status"`
	Missing    bool    `json:"missing"`
	ProjectID  *string `json:"project_id"`
	PlanID     *string `json:"plan_id"`
}

type aiMessageResponse struct {
	ID           string                 `json:"id"`
	Role         string                 `json:"role"`
	Content      string                 `json:"content"`
	Status       string                 `json:"status"`
	ErrorCode    *string                `json:"error_code"`
	ConnectionID *string                `json:"connection_id"`
	Model        *string                `json:"model"`
	CreatedAt    string                 `json:"created_at"`
	Attachments  []aiAttachmentResponse `json:"attachments"`
}

func toAIMessageResponse(m model.AIMessage) aiMessageResponse {
	return aiMessageResponse{
		ID: m.ID, Role: string(m.Role), Content: m.Content, Status: string(m.Status),
		ErrorCode: optionalString(m.ErrorCode), ConnectionID: optionalString(m.ConnectionID), Model: optionalString(m.Model),
		CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339), Attachments: []aiAttachmentResponse{},
	}
}

type aiChatDetailResponse struct {
	Chat     aiChatResponse      `json:"chat"`
	Messages []aiMessageResponse `json:"messages"`
}

func (h *AIHandler) status(w http.ResponseWriter, r *http.Request) {
	var active *string
	if id := h.chats.Status(); id != "" {
		active = &id
	}
	writeJSON(w, http.StatusOK, struct {
		ActiveChatID *string `json:"active_chat_id"`
	}{active})
}

func (h *AIHandler) listChats(w http.ResponseWriter, r *http.Request) {
	chats, err := h.chats.List(r.Context(), h.session, r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := make([]aiChatResponse, 0, len(chats))
	for _, c := range chats {
		res = append(res, toAIChatResponse(c))
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *AIHandler) getChat(w http.ResponseWriter, r *http.Request) {
	d, err := h.chats.Get(r.Context(), h.session, r.PathValue("id"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	res := aiChatDetailResponse{Chat: toAIChatResponse(d.Chat), Messages: make([]aiMessageResponse, 0, len(d.Messages))}
	for _, m := range d.Messages {
		mr := toAIMessageResponse(m.AIMessage)
		for _, a := range m.Attachments {
			mr.Attachments = append(mr.Attachments, aiAttachmentResponse{
				ID: a.ID, Kind: string(a.Kind), TargetID: a.TargetID, Path: optionalString(a.Path), Label: a.Label,
				LastStatus: string(a.LastStatus), Missing: a.Missing, ProjectID: optionalString(a.ProjectID), PlanID: optionalString(a.PlanID),
			})
		}
		res.Messages = append(res.Messages, mr)
	}
	writeJSON(w, http.StatusOK, res)
}

type aiAttachmentRequest struct {
	Kind     string `json:"kind"`
	TargetID string `json:"target_id"`
	Path     string `json:"path"`
}

type sendAIRequest struct {
	ConnectionID string                `json:"connection_id"`
	Model        string                `json:"model"`
	Content      string                `json:"content"`
	Attachments  []aiAttachmentRequest `json:"attachments"`
}

func (req sendAIRequest) toInput() service.SendAIInput {
	in := service.SendAIInput{ConnectionID: req.ConnectionID, Model: req.Model, Content: req.Content}
	for _, a := range req.Attachments {
		in.Attachments = append(in.Attachments, service.AIAttachmentInput{Kind: model.AIAttachmentKind(a.Kind), TargetID: a.TargetID, Path: a.Path})
	}
	return in
}

func (h *AIHandler) createChat(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	var req sendAIRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.chats.Create(r.Context(), h.session, req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, struct {
		Chat             aiChatResponse    `json:"chat"`
		UserMessage      aiMessageResponse `json:"user_message"`
		AssistantMessage aiMessageResponse `json:"assistant_message"`
	}{toAIChatResponse(res.Chat), toAIMessageResponse(res.UserMessage), toAIMessageResponse(res.AssistantMessage)})
}

func (h *AIHandler) send(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	var req sendAIRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	res, err := h.chats.Send(r.Context(), h.session, r.PathValue("id"), req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, struct {
		UserMessage      aiMessageResponse `json:"user_message"`
		AssistantMessage aiMessageResponse `json:"assistant_message"`
	}{toAIMessageResponse(res.UserMessage), toAIMessageResponse(res.AssistantMessage)})
}

type regenerateAIRequest struct {
	ConnectionID string `json:"connection_id"`
	Model        string `json:"model"`
}

func (h *AIHandler) regenerate(w http.ResponseWriter, r *http.Request) {
	if !requireJSON(w, r) {
		return
	}
	var req regenerateAIRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	m, err := h.chats.Regenerate(r.Context(), h.session, r.PathValue("id"), service.RegenerateAIInput{ConnectionID: req.ConnectionID, Model: req.Model})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, struct {
		AssistantMessage aiMessageResponse `json:"assistant_message"`
	}{toAIMessageResponse(m)})
}

type renameAIChatRequest struct {
	Title string `json:"title"`
}

func (h *AIHandler) renameChat(w http.ResponseWriter, r *http.Request) {
	var req renameAIChatRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := h.chats.Rename(r.Context(), h.session, r.PathValue("id"), req.Title)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toAIChatResponse(c))
}

func (h *AIHandler) deleteChat(w http.ResponseWriter, r *http.Request) {
	if err := h.chats.Delete(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AIHandler) stop(w http.ResponseWriter, r *http.Request) {
	if err := h.chats.Stop(r.Context(), h.session, r.PathValue("id")); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeSSE(w http.ResponseWriter, rc *http.ResponseController, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return err
	}
	return rc.Flush()
}

// stream sends the answer being written for a chat: what exists so far,
// then every new piece. Leaving does not stop the answer.
func (h *AIHandler) stream(w http.ResponseWriter, r *http.Request) {
	chatID := r.PathValue("id")
	if _, err := h.chats.Get(r.Context(), h.session, chatID); err != nil {
		writeError(w, r, err)
		return
	}
	st, ok := h.chats.Subscribe(chatID)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	rc := http.NewResponseController(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	messageID := st.MessageID()
	text, offset, changed, done, status := st.Read(0)
	if err := writeSSE(w, rc, "snapshot", map[string]string{"message_id": messageID, "content": text}); err != nil {
		return
	}
	ping := time.NewTicker(sseKeepAlive)
	defer ping.Stop()
	for {
		if done {
			_ = writeSSE(w, rc, "done", map[string]string{"message_id": messageID, "status": string(status)})
			return
		}
		select {
		case <-changed:
		case <-r.Context().Done():
			return
		case <-ping.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			if err := rc.Flush(); err != nil {
				return
			}
			continue
		}
		text, offset, changed, done, status = st.Read(offset)
		if text != "" {
			if err := writeSSE(w, rc, "delta", map[string]string{"text": text}); err != nil {
				return
			}
		}
	}
}
