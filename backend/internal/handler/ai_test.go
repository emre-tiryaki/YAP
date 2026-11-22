package handler

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/llm"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type memAIKeys struct {
	mu   sync.Mutex
	keys map[string]string
}

func (m *memAIKeys) Get(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if k, ok := m.keys[id]; ok {
		return k, nil
	}
	return "", model.ErrNoCredential
}

func (m *memAIKeys) Set(id, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[id] = key
	return nil
}

func (m *memAIKeys) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.keys, id)
	return nil
}

type fakeAILLM struct {
	mu     sync.Mutex
	deltas []string
	err    error
	gate   chan struct{}
	models []string
	// late are sent after the gate opens.
	late []string
}

func (f *fakeAILLM) setup(gate chan struct{}, err error, deltas ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gate, f.err, f.deltas = gate, err, deltas
}

func (f *fakeAILLM) Stream(ctx context.Context, req llm.Request, onDelta func(string)) error {
	f.mu.Lock()
	deltas, err, gate, late := f.deltas, f.err, f.gate, f.late
	f.mu.Unlock()
	for _, d := range deltas {
		onDelta(d)
	}
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for _, d := range late {
		onDelta(d)
	}
	return err
}

func (f *fakeAILLM) Models(context.Context, llm.Endpoint) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.models, f.err
}

type aiConnJSON struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Preset  string `json:"preset"`
	BaseURL string `json:"base_url"`
	HasKey  bool   `json:"has_key"`
}

func (e testEnv) aiConnection(t *testing.T) aiConnJSON {
	t.Helper()
	rec := e.do(t, http.MethodPost, "/api/ai/connections", `{"name":"Local","preset":"ollama","base_url":"http://localhost:11434/v1"}`)
	wantStatus(t, rec, http.StatusCreated)
	return decode[aiConnJSON](t, rec)
}

type aiSendJSON struct {
	Chat struct {
		ID         string `json:"id"`
		Title      string `json:"title"`
		Generating bool   `json:"generating"`
	} `json:"chat"`
	UserMessage      struct{ ID, Content string } `json:"user_message"`
	AssistantMessage struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	} `json:"assistant_message"`
}

func (e testEnv) aiChat(t *testing.T, connID, content string) aiSendJSON {
	t.Helper()
	rec := e.do(t, http.MethodPost, "/api/ai/chats", `{"connection_id":"`+connID+`","model":"m1","content":"`+content+`","attachments":[]}`)
	wantStatus(t, rec, http.StatusAccepted)
	return decode[aiSendJSON](t, rec)
}

// waitAIIdle blocks until the running generation has written its final
// state. It waits on the service's work group instead of polling, and fails
// rather than hangs when the model's gate was never opened.
func (e testEnv) waitAIIdle(t *testing.T) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		e.waitAI()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("generation did not finish")
	}
}

func TestAIConnectionEndpoints(t *testing.T) {
	e := newTestEnv(t)
	c := e.aiConnection(t)
	if c.Name != "Local" || c.HasKey || c.Preset != "ollama" {
		t.Errorf("connection = %+v", c)
	}

	// Key is stored in the keychain, never returned.
	rec := e.do(t, http.MethodPost, "/api/ai/connections", `{"name":"OpenAI","preset":"openai","base_url":"https://api.openai.com/v1","api_key":"sk-secret"}`)
	wantStatus(t, rec, http.StatusCreated)
	if strings.Contains(rec.Body.String(), "sk-secret") {
		t.Errorf("response leaks the key: %s", rec.Body.String())
	}
	keyed := decode[aiConnJSON](t, rec)
	if !keyed.HasKey || e.keys.keys[keyed.ID] != "sk-secret" {
		t.Errorf("keyed = %+v, stored %q", keyed, e.keys.keys[keyed.ID])
	}

	list := decode[[]aiConnJSON](t, e.do(t, http.MethodGet, "/api/ai/connections", ""))
	if len(list) != 2 {
		t.Errorf("list = %+v", list)
	}
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/connections", `{"name":"x","preset":"openai","base_url":"https://api.openai.com/v1"}`), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/connections", `{"name":"","preset":"ollama","base_url":"http://x"}`), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/connections", `{"name":"x","preset":"ollama","base_url":"http://x","extra":1}`), http.StatusBadRequest, "invalid_input")

	rec = e.do(t, http.MethodPatch, "/api/ai/connections/"+keyed.ID, `{"name":"Work","api_key":"sk-2"}`)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[aiConnJSON](t, rec); got.Name != "Work" || e.keys.keys[keyed.ID] != "sk-2" {
		t.Errorf("updated = %+v", got)
	}
	wantErrorCode(t, e.do(t, http.MethodPatch, "/api/ai/connections/"+keyed.ID, `{"api_key":""}`), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.do(t, http.MethodPatch, "/api/ai/connections/nope", `{"name":"x"}`), http.StatusNotFound, "not_found")

	rec = e.do(t, http.MethodGet, "/api/ai/connections/"+c.ID+"/models", "")
	wantStatus(t, rec, http.StatusOK)
	if got := decode[struct{ Models []string }](t, rec).Models; len(got) != 2 {
		t.Errorf("models = %v", got)
	}

	wantStatus(t, e.do(t, http.MethodDelete, "/api/ai/connections/"+keyed.ID, ""), http.StatusNoContent)
	if _, ok := e.keys.keys[keyed.ID]; ok {
		t.Error("key survived delete")
	}
	wantErrorCode(t, e.do(t, http.MethodGet, "/api/ai/connections/"+keyed.ID+"/models", ""), http.StatusConflict, "ai_connection_missing")
}

func TestAIConnectionTestEndpoint(t *testing.T) {
	e := newTestEnv(t)
	body := `{"preset":"ollama","base_url":"http://localhost:11434/v1"}`
	rec := e.do(t, http.MethodPost, "/api/ai/connections/test", body)
	wantStatus(t, rec, http.StatusOK)
	if got := decode[struct{ Models []string }](t, rec).Models; len(got) != 2 {
		t.Errorf("models = %v", got)
	}
	// Needs a JSON content type.
	req := httptest.NewRequest(http.MethodPost, "/api/ai/connections/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	e.mux.ServeHTTP(rr, req)
	wantErrorCode(t, rr, http.StatusUnsupportedMediaType, "invalid_input")

	for err, want := range map[error]struct {
		status int
		code   string
	}{
		service.ErrAIUnreachable:   {http.StatusBadGateway, "ai_unreachable"},
		service.ErrAIAuthFailed:    {http.StatusBadGateway, "ai_auth_failed"},
		service.ErrAIRateLimited:   {http.StatusTooManyRequests, "ai_rate_limited"},
		service.ErrAIModelNotFound: {http.StatusBadGateway, "ai_model_not_found"},
		service.ErrAIRejected:      {http.StatusBadGateway, "ai_rejected"},
	} {
		e.llm.setup(nil, err)
		wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/connections/test", body), want.status, want.code)
	}
}

func TestAIChatEndpoints(t *testing.T) {
	e := newTestEnv(t)
	c := e.aiConnection(t)
	e.llm.setup(nil, nil, "Hello", " world")

	res := e.aiChat(t, c.ID, "Sprint summary")
	if res.Chat.Title != "Sprint summary" || res.AssistantMessage.Status != "streaming" || res.UserMessage.Content != "Sprint summary" {
		t.Errorf("create = %+v", res)
	}
	e.waitAIIdle(t)

	rec := e.do(t, http.MethodGet, "/api/ai/chats/"+res.Chat.ID, "")
	wantStatus(t, rec, http.StatusOK)
	d := decode[struct {
		Chat     struct{ ID, Model string }
		Messages []struct {
			Role, Content, Status string
			ConnectionID          *string `json:"connection_id"`
			Attachments           []any
		}
	}](t, rec)
	if len(d.Messages) != 2 || d.Messages[1].Content != "Hello world" || d.Messages[1].Status != "complete" ||
		d.Messages[1].ConnectionID == nil || d.Messages[0].Attachments == nil {
		t.Errorf("detail = %+v", d)
	}

	// Second message, regenerate, rename, search, delete.
	wantStatus(t, e.do(t, http.MethodPost, "/api/ai/chats/"+res.Chat.ID+"/messages", `{"connection_id":"`+c.ID+`","model":"m1","content":"more"}`), http.StatusAccepted)
	e.waitAIIdle(t)
	rec = e.do(t, http.MethodPost, "/api/ai/chats/"+res.Chat.ID+"/regenerate", `{"connection_id":"`+c.ID+`","model":"m2"}`)
	wantStatus(t, rec, http.StatusAccepted)
	e.waitAIIdle(t)

	rec = e.do(t, http.MethodPatch, "/api/ai/chats/"+res.Chat.ID, `{"title":"Renamed"}`)
	wantStatus(t, rec, http.StatusOK)
	wantErrorCode(t, e.do(t, http.MethodPatch, "/api/ai/chats/"+res.Chat.ID, `{"title":" "}`), http.StatusBadRequest, "invalid_input")
	list := decode[[]struct{ ID, Title string }](t, e.do(t, http.MethodGet, "/api/ai/chats?q=renam", ""))
	if len(list) != 1 || list[0].Title != "Renamed" {
		t.Errorf("search = %+v", list)
	}
	if list := decode[[]struct{ ID string }](t, e.do(t, http.MethodGet, "/api/ai/chats?q=zzz", "")); len(list) != 0 {
		t.Errorf("search zzz = %+v", list)
	}

	wantStatus(t, e.do(t, http.MethodDelete, "/api/ai/chats/"+res.Chat.ID, ""), http.StatusNoContent)
	wantErrorCode(t, e.do(t, http.MethodGet, "/api/ai/chats/"+res.Chat.ID, ""), http.StatusNotFound, "not_found")
}

func TestAIChatValidationAndContentType(t *testing.T) {
	e := newTestEnv(t)
	c := e.aiConnection(t)
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/chats", `{"connection_id":"`+c.ID+`","model":"m1","content":" "}`), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/chats", `{"connection_id":"nope","model":"m1","content":"hi"}`), http.StatusConflict, "ai_connection_missing")
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/chats", `{"connection_id":"`+c.ID+`","model":"m1","content":"hi","attachments":[{"kind":"task","target_id":"nope"}]}`), http.StatusNotFound, "not_found")

	for _, path := range []string{"/api/ai/chats", "/api/ai/chats/x/messages", "/api/ai/chats/x/regenerate"} {
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()
		e.mux.ServeHTTP(rec, req)
		wantErrorCode(t, rec, http.StatusUnsupportedMediaType, "invalid_input")
	}
}

func TestAIBusyAndStop(t *testing.T) {
	e := newTestEnv(t)
	c := e.aiConnection(t)
	gate := make(chan struct{})
	e.llm.setup(gate, nil, "part")
	res := e.aiChat(t, c.ID, "first")

	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/chats", `{"connection_id":"`+c.ID+`","model":"m1","content":"second"}`), http.StatusConflict, "ai_busy")
	st := decode[struct {
		ActiveChatID *string `json:"active_chat_id"`
	}](t, e.do(t, http.MethodGet, "/api/ai/status", ""))
	if st.ActiveChatID == nil || *st.ActiveChatID != res.Chat.ID {
		t.Errorf("status = %+v", st)
	}

	wantStatus(t, e.do(t, http.MethodPost, "/api/ai/chats/"+res.Chat.ID+"/stop", ""), http.StatusNoContent)
	d := decode[struct {
		Messages []struct{ Content, Status string }
	}](t, e.do(t, http.MethodGet, "/api/ai/chats/"+res.Chat.ID, ""))
	if m := d.Messages[1]; m.Status != "stopped" || m.Content != "part" {
		t.Errorf("assistant = %+v", m)
	}
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/ai/chats/nope/stop", ""), http.StatusNotFound, "not_found")
}

func TestAIStreamEndpoint(t *testing.T) {
	e := newTestEnv(t)
	c := e.aiConnection(t)

	// Idle chat: 204.
	e.llm.setup(nil, nil, "x")
	idle := e.aiChat(t, c.ID, "idle")
	e.waitAIIdle(t)
	wantStatus(t, e.do(t, http.MethodGet, "/api/ai/chats/"+idle.Chat.ID+"/stream", ""), http.StatusNoContent)
	wantErrorCode(t, e.do(t, http.MethodGet, "/api/ai/chats/nope/stream", ""), http.StatusNotFound, "not_found")

	gate := make(chan struct{})
	e.llm.setup(gate, nil, "ab")
	e.llm.late = []string{"cd"}
	res := e.aiChat(t, c.ID, "go")

	srv := httptest.NewServer(e.mux)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/api/ai/chats/" + res.Chat.ID + "/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}
	lines := make(chan string, 32)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()
	next := func() string {
		select {
		case l, ok := <-lines:
			if !ok {
				return "<closed>"
			}
			return l
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for the stream")
			return ""
		}
	}
	if l := next(); l != "event: snapshot" {
		t.Fatalf("first line = %q", l)
	}
	if l := next(); !strings.Contains(l, `"content":"ab"`) || !strings.Contains(l, res.AssistantMessage.ID) {
		t.Errorf("snapshot data = %q", l)
	}
	next() // blank line
	close(gate)
	var sawDone, sawDelta bool
	for l := next(); l != "<closed>"; l = next() {
		if l == "event: delta" {
			sawDelta = true
			if d := next(); d != `data: {"text":"cd"}` {
				t.Errorf("delta data = %q", d)
			}
		}
		if l == "event: done" {
			sawDone = true
			if d := next(); !strings.Contains(d, `"status":"complete"`) {
				t.Errorf("done data = %q", d)
			}
		}
	}
	if !sawDone || !sawDelta {
		t.Errorf("sawDelta = %v, sawDone = %v", sawDelta, sawDone)
	}
}

func TestResetBodyIsOptional(t *testing.T) {
	e := newTestEnv(t)
	wantStatus(t, e.do(t, http.MethodPost, "/api/account/reset", ""), http.StatusNoContent)
	wantStatus(t, e.do(t, http.MethodPost, "/api/account/reset", `{"ai_chats":true,"ai_connections":false}`), http.StatusNoContent)
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/account/reset", `{"nope":true}`), http.StatusBadRequest, "invalid_input")
	wantErrorCode(t, e.do(t, http.MethodPost, "/api/account/reset", `{`), http.StatusBadRequest, "invalid_input")
}
