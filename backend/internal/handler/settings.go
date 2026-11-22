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

type statusNamesResponse struct {
	ToDo       string `json:"todo"`
	InProgress string `json:"in_progress"`
	Done       string `json:"done"`
}

type notificationSettingsResponse struct {
	DueSoon  bool   `json:"due_soon"`
	Reminder string `json:"reminder"`
	Overdue  bool   `json:"overdue"`
	Comments bool   `json:"comments"`
	Sound    bool   `json:"sound"`
}

type settingsResponse struct {
	AutosaveEnabled bool                         `json:"autosave_enabled"`
	AutosaveDelayMs int                          `json:"autosave_delay_ms"`
	HideDone        bool                         `json:"hide_done"`
	DoneToBottom    bool                         `json:"done_to_bottom"`
	StatusNames     statusNamesResponse          `json:"status_names"`
	Notifications   notificationSettingsResponse `json:"notifications"`
	RepoAutoUpdate  bool                         `json:"repositories_auto_update"`
	AIInstructions  string                       `json:"ai_instructions"`
}

type updateNotificationSettingsRequest struct {
	DueSoon  *bool   `json:"due_soon"`
	Reminder *string `json:"reminder"`
	Overdue  *bool   `json:"overdue"`
	Comments *bool   `json:"comments"`
	Sound    *bool   `json:"sound"`
}

type updateSettingsRequest struct {
	AutosaveEnabled *bool `json:"autosave_enabled"`
	AutosaveDelayMs *int  `json:"autosave_delay_ms"`
	HideDone        *bool `json:"hide_done"`
	DoneToBottom    *bool `json:"done_to_bottom"`
	// StatusNames maps a status category to its new default name.
	StatusNames    map[string]string                  `json:"status_names"`
	Notifications  *updateNotificationSettingsRequest `json:"notifications"`
	RepoAutoUpdate *bool                              `json:"repositories_auto_update"`
	AIInstructions *string                            `json:"ai_instructions"`
}

func toSettingsResponse(s service.UserSettings) settingsResponse {
	n := s.Notifications
	return settingsResponse{
		AutosaveEnabled: s.AutosaveEnabled,
		AutosaveDelayMs: s.AutosaveDelayMs,
		HideDone:        s.HideDone,
		DoneToBottom:    s.DoneToBottom,
		StatusNames: statusNamesResponse{
			ToDo:       s.StatusNames.ToDo,
			InProgress: s.StatusNames.InProgress,
			Done:       s.StatusNames.Done,
		},
		Notifications: notificationSettingsResponse{
			DueSoon:  n.DueSoon,
			Reminder: string(n.Reminder),
			Overdue:  n.Overdue,
			Comments: n.Comments,
			Sound:    n.Sound,
		},
		RepoAutoUpdate: s.RepoAutoUpdate,
		AIInstructions: s.AIInstructions,
	}
}

func (req updateSettingsRequest) toInput() service.UpdateSettingsInput {
	in := service.UpdateSettingsInput{
		AutosaveEnabled: req.AutosaveEnabled,
		AutosaveDelayMs: req.AutosaveDelayMs,
		HideDone:        req.HideDone,
		DoneToBottom:    req.DoneToBottom,
		RepoAutoUpdate:  req.RepoAutoUpdate,
		AIInstructions:  req.AIInstructions,
	}
	if len(req.StatusNames) > 0 {
		in.StatusNames = map[model.StatusCategory]string{}
		for c, name := range req.StatusNames {
			in.StatusNames[model.StatusCategory(c)] = name
		}
	}
	if n := req.Notifications; n != nil {
		in.NotifyDueSoon = n.DueSoon
		in.NotifyOverdue = n.Overdue
		in.NotifyComments = n.Comments
		in.NotifySound = n.Sound
		if n.Reminder != nil {
			r := service.Reminder(*n.Reminder)
			in.NotifyReminder = &r
		}
	}
	return in
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
	s, err := h.svc.Update(r.Context(), h.session, req.toInput())
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toSettingsResponse(s))
}
