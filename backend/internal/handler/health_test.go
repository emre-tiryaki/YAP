package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHelloWorld(t *testing.T) {
	rec := httptest.NewRecorder()
	HelloWorld(rec, httptest.NewRequest(http.MethodGet, "/hello-world", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status=%d, want 200", rec.Code)
	}
	if got := rec.Body.String(); got != "Server is working!!!" {
		t.Errorf("body=%q", got)
	}
}