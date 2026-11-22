package service

import (
	"context"
	"sync"

	"github.com/emre-tiryaki/YAP/backend/internal/llm"
	"github.com/emre-tiryaki/YAP/backend/internal/model"
)

// memAIKeys is an in-memory AIKeyStore.
type memAIKeys struct {
	mu   sync.Mutex
	keys map[string]string
	// err, when set, is returned by every call.
	err error
}

func newMemAIKeys() *memAIKeys { return &memAIKeys{keys: map[string]string{}} }

func (m *memAIKeys) Get(id string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return "", m.err
	}
	k, ok := m.keys[id]
	if !ok {
		return "", model.ErrNoCredential
	}
	return k, nil
}

func (m *memAIKeys) Set(id, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.keys[id] = key
	return nil
}

func (m *memAIKeys) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	delete(m.keys, id)
	return nil
}

// fakeLLM answers with scripted deltas.
type fakeLLM struct {
	mu     sync.Mutex
	deltas []string
	err    error
	// gate, when set, holds the stream open after the deltas until it is
	// closed or the context ends.
	gate chan struct{}
	// started is closed when the first Stream call begins (if set).
	started chan struct{}
	models  []string

	requests []llm.Request
	endpoint []llm.Endpoint
}

func (f *fakeLLM) Stream(ctx context.Context, req llm.Request, onDelta func(string)) error {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	deltas, err, gate, started := f.deltas, f.err, f.gate, f.started
	f.started = nil
	f.mu.Unlock()
	if started != nil {
		close(started)
	}
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
	return err
}

func (f *fakeLLM) Models(ctx context.Context, ep llm.Endpoint) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.endpoint = append(f.endpoint, ep)
	if f.err != nil {
		return nil, f.err
	}
	return f.models, nil
}

func (f *fakeLLM) lastRequest() llm.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.requests[len(f.requests)-1]
}
