package handler

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAccountExport(t *testing.T) {
	env := newTestEnv(t)
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha"}`), http.StatusCreated)

	rec := env.do(t, http.MethodGet, "/api/account/export", "")

	wantStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "application/zip" {
		t.Errorf("Content-Type = %q, want application/zip", got)
	}
	if got := rec.Header().Get("Content-Disposition"); !strings.HasPrefix(got, `attachment; filename="yap-export-`) {
		t.Errorf("Content-Disposition = %q", got)
	}
	body := rec.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}
	found := false
	for _, f := range zr.File {
		found = found || f.Name == "data.json"
	}
	if !found {
		t.Error("data.json missing from the archive")
	}
}

func TestAccountReset(t *testing.T) {
	env := newTestEnv(t)
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha"}`), http.StatusCreated)

	wantStatus(t, env.do(t, http.MethodPost, "/api/account/reset", ""), http.StatusNoContent)

	rec := env.do(t, http.MethodGet, "/api/projects", "")
	wantStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("projects after reset = %s, want []", got)
	}
}

func TestAccountImport(t *testing.T) {
	env := newTestEnv(t)
	wantStatus(t, env.do(t, http.MethodPost, "/api/projects", `{"name":"Alpha"}`), http.StatusCreated)
	archive := env.do(t, http.MethodGet, "/api/account/export", "").Body.Bytes()
	wantStatus(t, env.do(t, http.MethodPost, "/api/account/reset", ""), http.StatusNoContent)

	t.Run("restores an export", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/account/import", bytes.NewReader(archive))
		req.Header.Set("Content-Type", "application/zip")
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)

		wantStatus(t, rec, http.StatusOK)
		got := decode[importResponse](t, rec)
		if !got.Restored || got.Added.Projects != 1 || got.MissingFiles == nil {
			t.Errorf("response = %+v, want a restore of one project", got)
		}
		rec = env.do(t, http.MethodGet, "/api/projects", "")
		if !strings.Contains(rec.Body.String(), `"Alpha"`) {
			t.Errorf("projects = %s, want Alpha", rec.Body.String())
		}
	})

	t.Run("rejects something else", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/account/import", strings.NewReader("not a zip"))
		rec := httptest.NewRecorder()
		env.mux.ServeHTTP(rec, req)
		wantErrorCode(t, rec, http.StatusBadRequest, "invalid_input")
	})
}
