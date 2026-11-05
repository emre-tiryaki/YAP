package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]string{"k": "v"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
	if got := decode[map[string]string](t, rec); got["k"] != "v" {
		t.Errorf("body = %v", got)
	}
}

func TestWriteError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
	}{
		{"invalid input", service.ErrInvalidInput, http.StatusBadRequest, "invalid_input"},
		{"wrapped invalid input", fmt.Errorf("%w: name is required", service.ErrInvalidInput), http.StatusBadRequest, "invalid_input"},
		{"not found", service.ErrNotFound, http.StatusNotFound, "not_found"},
		{"wrapped not found", fmt.Errorf("%w: project", service.ErrNotFound), http.StatusNotFound, "not_found"},
		{"unknown error", errors.New("db exploded"), http.StatusInternalServerError, "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			writeError(rec, req, tt.err)

			wantErrorCode(t, rec, tt.wantStatus, tt.wantCode)
			if strings.Contains(rec.Body.String(), "db exploded") {
				t.Errorf("body leaks internal error text: %q", rec.Body.String())
			}
		})
	}
}

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name    string
		body    string
		want    string
		wantErr bool
	}{
		{"valid", `{"name":"x"}`, "x", false},
		{"malformed", `{"name":`, "", true},
		{"unknown field", `{"name":"x","extra":1}`, "", true},
		{"wrong type", `{"name":5}`, "", true},
		{"empty body", ``, "", true},
		{"too large", `{"name":"` + strings.Repeat("a", maxBodyBytes+10) + `"}`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(tt.body))

			var p payload
			err := decodeJSON(rec, req, &p)
			if (err != nil) != tt.wantErr {
				t.Fatalf("decodeJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, service.ErrInvalidInput) {
				t.Errorf("error = %v, want ErrInvalidInput", err)
			}
			if p.Name != tt.want {
				t.Errorf("Name = %q, want %q", p.Name, tt.want)
			}
		})
	}
}