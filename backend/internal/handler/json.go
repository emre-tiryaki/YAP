package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

const maxBodyBytes = 1 << 20

type errorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("Encode response", "error", err)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	var maxBytes *http.MaxBytesError
	switch {
	case errors.Is(err, service.ErrTooLarge), errors.As(err, &maxBytes):
		writeJSON(w, http.StatusRequestEntityTooLarge, errorResponse{Error: "too_large"})
	case errors.Is(err, service.ErrStale):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "stale"})
	case errors.Is(err, service.ErrFileMissing):
		writeJSON(w, http.StatusGone, errorResponse{Error: "file_missing"})
	case errors.Is(err, service.ErrReadOnly):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "read_only"})
	case errors.Is(err, service.ErrGitHubNotConfigured):
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "github_not_configured"})
	case errors.Is(err, service.ErrGitHubClientInvalid):
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "github_client_invalid"})
	case errors.Is(err, service.ErrDeviceFlowDisabled):
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "github_device_flow_disabled"})
	case errors.Is(err, service.ErrGitHubNotConnected):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "github_not_connected"})
	case errors.Is(err, service.ErrGitHubUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "github_unavailable"})
	case errors.Is(err, service.ErrKeychainUnavailable):
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "keychain_unavailable"})
	case errors.Is(err, service.ErrGitMissing):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "git_missing"})
	case errors.Is(err, service.ErrInvalidInput):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid_input"})
	case errors.Is(err, service.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not_found"})
	case errors.Is(err, service.ErrConflict):
		writeJSON(w, http.StatusConflict, errorResponse{Error: "conflict"})
	case errors.Is(err, service.ErrForbidden):
		writeJSON(w, http.StatusForbidden, errorResponse{Error: "forbidden"})
	default:
		slog.Error("request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "internal"})
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("%w: malformed body", service.ErrInvalidInput)
	}
	return nil
}
