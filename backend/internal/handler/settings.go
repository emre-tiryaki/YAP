package handler

import (
	"net/http"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type SettingsHandler struct {
	session model.Session
	svc     *service.SettingsService
}

func NewSettingsHandler(session model.Session, svc *service.SettingsService) *SettingsHandler {
	return &SettingsHandler{session: session, svc: svc}
}

func (h *SettingsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/settings", h.get)
	mux.HandleFunc("PATCH /api/settings", h.update)
}

type settingsResponse struct {
	AutosaveEnabled bool `json:"autosave_enabled"`
	AutosaveDelayMs int  `json:"autosave_delay_ms"`
}

type updateSettingsRequest struct {
	AutosaveEnabled *bool `json:"autosave_enabled"`
	AutosaveDelayMs *int  `json:"autosave_delay_ms"`
}

func toSettingsResponse(s service.UserSettings) settingsResponse {
	return settingsResponse{AutosaveEnabled: s.AutosaveEnabled, AutosaveDelayMs: s.AutosaveDelayMs}
}

func (h *SettingsHandler) get(w http.ResponseWriter, r *http.Request) {
	s, err := h.svc.Get(r.Context(), h.session)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSettingsResponse(s))
}

func (h *SettingsHandler) update(w http.ResponseWriter, r *http.Request) {
	var req updateSettingsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	s, err := h.svc.Update(r.Context(), h.session, service.UpdateSettingsInput{
		AutosaveEnabled: req.AutosaveEnabled,
		AutosaveDelayMs: req.AutosaveDelayMs,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSettingsResponse(s))
}
