package llm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

func openAISSE(w http.ResponseWriter, pieces ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	for _, p := range pieces {
		fmt.Fprintf(w, "data: {\"id\":\"x\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":%q}}]}\n\n", p)
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
}

func collect() (*[]string, func(string)) {
	var got []string
	return &got, func(s string) { got = append(got, s) }
}

func TestStreamOpenAI(t *testing.T) {
	var auth string
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		b := new(strings.Builder)
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		b.Write(buf[:n])
		body = b.String()
		openAISSE(w, "He", "llo")
	}))
	defer srv.Close()

	got, onDelta := collect()
	err := NewRouter().Stream(context.Background(), Request{
		Endpoint: Endpoint{Provider: ProviderOpenAICompatible, BaseURL: srv.URL + "/v1"},
		Model:    "m1", System: "sys",
		Messages: []Message{{RoleUser, "hi"}, {RoleAssistant, "yo"}, {RoleUser, "again"}},
	}, onDelta)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if !reflect.DeepEqual(*got, []string{"He", "llo"}) {
		t.Errorf("deltas = %v", *got)
	}
	if auth != "Bearer "+noKey {
		t.Errorf("Authorization = %q, want placeholder key", auth)
	}
	for _, want := range []string{`"sys"`, `"yo"`, `"again"`, `"m1"`} {
		if !strings.Contains(body, want) {
			t.Errorf("request body lacks %s: %s", want, body)
		}
	}
}

func TestOpenAIIgnoresEnvironment(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "env-key")
	t.Setenv("OPENAI_BASE_URL", "http://127.0.0.1:1/never")
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		openAISSE(w, "ok")
	}))
	defer srv.Close()
	_, onDelta := collect()
	err := NewRouter().Stream(context.Background(), Request{
		Endpoint: Endpoint{Provider: ProviderOpenAICompatible, BaseURL: srv.URL + "/v1"},
		Model:    "m", Messages: []Message{{RoleUser, "x"}},
	}, onDelta)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if auth != "Bearer "+noKey {
		t.Errorf("Authorization = %q", auth)
	}
}

func anthropicSSE(w http.ResponseWriter, pieces ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	fmt.Fprint(w, "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"m\",\"type\":\"message\",\"role\":\"assistant\",\"content\":[],\"model\":\"c\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}}\n\n")
	fmt.Fprint(w, "event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n")
	for _, p := range pieces {
		fmt.Fprintf(w, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":%q}}\n\n", p)
	}
	fmt.Fprint(w, "event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n")
	fmt.Fprint(w, "event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
}

func TestStreamAnthropic(t *testing.T) {
	var key, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key = r.Header.Get("X-Api-Key")
		buf := make([]byte, 8192)
		n, _ := r.Body.Read(buf)
		body = string(buf[:n])
		anthropicSSE(w, "Hel", "lo")
	}))
	defer srv.Close()
	got, onDelta := collect()
	err := NewRouter().Stream(context.Background(), Request{
		Endpoint: Endpoint{Provider: ProviderAnthropic, BaseURL: srv.URL, APIKey: "sk-test"},
		Model:    "claude-x", System: "sys",
		// Two adjacent user turns must be merged.
		Messages: []Message{{RoleUser, "a"}, {RoleUser, "b"}},
	}, onDelta)
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if !reflect.DeepEqual(*got, []string{"Hel", "lo"}) {
		t.Errorf("deltas = %v", *got)
	}
	if key != "sk-test" {
		t.Errorf("x-api-key = %q", key)
	}
	if !strings.Contains(body, `a\n\nb`) {
		t.Errorf("adjacent user messages not merged: %s", body)
	}
}

func TestAnthropicMessagesMerge(t *testing.T) {
	got := anthropicMessages([]Message{{RoleUser, "a"}, {RoleUser, "b"}, {RoleAssistant, "c"}, {RoleUser, "d"}})
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
}

func TestModels(t *testing.T) {
	t.Run("openai sorted", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/models" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"object":"list","data":[{"id":"b","object":"model","created":1,"owned_by":"x"},{"id":"a","object":"model","created":1,"owned_by":"x"}]}`)
		}))
		defer srv.Close()
		ids, err := NewRouter().Models(context.Background(), Endpoint{Provider: ProviderOpenAICompatible, BaseURL: srv.URL + "/v1"})
		if err != nil {
			t.Fatalf("Models() error = %v", err)
		}
		if !reflect.DeepEqual(ids, []string{"a", "b"}) {
			t.Errorf("ids = %v", ids)
		}
	})
	t.Run("anthropic", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"data":[{"id":"claude-b","type":"model","display_name":"B","created_at":"2026-01-01T00:00:00Z"},{"id":"claude-a","type":"model","display_name":"A","created_at":"2026-01-01T00:00:00Z"}],"has_more":false,"first_id":"claude-b","last_id":"claude-a"}`)
		}))
		defer srv.Close()
		ids, err := NewRouter().Models(context.Background(), Endpoint{Provider: ProviderAnthropic, BaseURL: srv.URL, APIKey: "k"})
		if err != nil {
			t.Fatalf("Models() error = %v", err)
		}
		if !reflect.DeepEqual(ids, []string{"claude-a", "claude-b"}) {
			t.Errorf("ids = %v", ids)
		}
	})
}

func TestErrorMapping(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{401, model.ErrAIAuthFailed},
		{403, model.ErrAIAuthFailed},
		{404, model.ErrAIModelNotFound},
		{429, model.ErrAIRateLimited},
		{400, model.ErrAIRejected},
		{500, model.ErrAIUnreachable},
	}
	for _, provider := range []Provider{ProviderOpenAICompatible, ProviderAnthropic} {
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s/%d", provider, tt.status), func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.status)
					fmt.Fprint(w, `{"error":{"message":"nope","type":"x"}}`)
				}))
				defer srv.Close()
				base := srv.URL
				if provider == ProviderOpenAICompatible {
					base += "/v1"
				}
				ep := Endpoint{Provider: provider, BaseURL: base, APIKey: "k"}
				_, onDelta := collect()
				err := NewRouter().Stream(context.Background(), Request{Endpoint: ep, Model: "m", Messages: []Message{{RoleUser, "x"}}}, onDelta)
				if !errors.Is(err, tt.want) {
					t.Errorf("Stream() error = %v, want %v", err, tt.want)
				}
				_, err = NewRouter().Models(context.Background(), ep)
				if !errors.Is(err, tt.want) {
					t.Errorf("Models() error = %v, want %v", err, tt.want)
				}
			})
		}
	}
}

func TestConnectionRefusedIsUnreachable(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	_, onDelta := collect()
	err = NewRouter().Stream(context.Background(), Request{
		Endpoint: Endpoint{Provider: ProviderOpenAICompatible, BaseURL: "http://" + addr + "/v1"},
		Model:    "m", Messages: []Message{{RoleUser, "x"}},
	}, onDelta)
	if !errors.Is(err, model.ErrAIUnreachable) {
		t.Errorf("Stream() error = %v, want unreachable", err)
	}
}

func TestStreamCancelMidStream(t *testing.T) {
	release := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"id\":\"x\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"a\"}}]}\n\n")
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer srv.Close()
	defer once.Do(func() { close(release) })

	ctx, cancel := context.WithCancel(context.Background())
	err := NewRouter().Stream(ctx, Request{
		Endpoint: Endpoint{Provider: ProviderOpenAICompatible, BaseURL: srv.URL + "/v1"},
		Model:    "m", Messages: []Message{{RoleUser, "x"}},
	}, func(string) { cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("Stream() error = %v, want context.Canceled", err)
	}
}
