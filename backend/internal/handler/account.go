package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/emre-tiryaki/YAP/backend/internal/model"
	"github.com/emre-tiryaki/YAP/backend/internal/service"
)

type AccountHandler struct {
	session model.Session
	svc     *service.AccountService
}

func NewAccountHandler(session model.Session, svc *service.AccountService) *AccountHandler {
	return &AccountHandler{session: session, svc: svc}
}

func (h *AccountHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/account/export", h.export)
	mux.HandleFunc("POST /api/account/import", h.importArchive)
	mux.HandleFunc("POST /api/account/reset", h.reset)
}

type importCountsResponse struct {
	Projects  int `json:"projects"`
	Plans     int `json:"plans"`
	Tasks     int `json:"tasks"`
	Comments  int `json:"comments"`
	Resources int `json:"resources"`
}

type importResponse struct {
	Restored     bool                 `json:"restored"`
	Added        importCountsResponse `json:"added"`
	Existing     int                  `json:"existing"`
	Renamed      int                  `json:"renamed"`
	MissingFiles []string             `json:"missing_files"`
}

// importArchive takes an archive made by the export as the raw body.
func (h *AccountHandler) importArchive(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, service.MaxImportBytes+1)
	res, err := h.svc.Import(r.Context(), h.session, body)
	if err != nil {
		writeError(w, r, err)
		return
	}
	missing := res.MissingFiles
	if missing == nil {
		missing = []string{}
	}
	writeJSON(w, http.StatusOK, importResponse{
		Restored: res.Restored,
		Added: importCountsResponse{
			Projects: res.Added.Projects, Plans: res.Added.Plans, Tasks: res.Added.Tasks,
			Comments: res.Added.Comments, Resources: res.Added.Resources,
		},
		Existing:     res.Existing,
		Renamed:      res.Renamed,
		MissingFiles: missing,
	})
}

func (h *AccountHandler) export(w http.ResponseWriter, r *http.Request) {
	name := "yap-export-" + time.Now().UTC().Format("20060102-150405") + ".zip"
	aw := &attachmentWriter{w: w, contentType: "application/zip", filename: name}
	if err := h.svc.Export(r.Context(), h.session, aw); err != nil {
		if !aw.started {
			writeError(w, r, err)
			return
		}
		// The archive is already partly sent; the client sees it cut off.
		slog.Error("export failed", "error", err)
	}
}

func (h *AccountHandler) reset(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Reset(r.Context(), h.session); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// attachmentWriter sends the download headers with the first byte, so an
// error before that can still be answered as JSON.
type attachmentWriter struct {
	w           http.ResponseWriter
	contentType string
	filename    string
	started     bool
}

func (a *attachmentWriter) Write(p []byte) (int, error) {
	if !a.started {
		a.started = true
		a.w.Header().Set("Content-Type", a.contentType)
		a.w.Header().Set("Content-Disposition", `attachment; filename="`+a.filename+`"`)
		a.w.WriteHeader(http.StatusOK)
	}
	return a.w.Write(p)
}
