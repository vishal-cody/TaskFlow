package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/vishalyadav/jobplatform/internal/handler"
)

// envelope mirrors the response package's envelope struct for test deserialization.
type envelope struct {
	Data json.RawMessage `json:"data"`
}

// testhealthlive verifies the liveness probe returns 200 with {"data":{"status":"ok"}}.
func TestHealthLive(t *testing.T) {
	healthHandler := handler.NewHealthHandler(nil, nil)

	r := chi.NewRouter()
	r.Get("/health/live", healthHandler.Live)

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// the response helper wraps in {"data": ...}
	var env envelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("failed to parse envelope: %v", err)
	}

	var body map[string]string
	if err := json.Unmarshal(env.Data, &body); err != nil {
		t.Fatalf("failed to parse data: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", body["status"])
	}
}

// testauthregister_invalidbody verifies that a malformed request body returns 400.
func TestAuthRegister_InvalidBody(t *testing.T) {
	authHandler := handler.NewAuthHandler(nil)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register",
		strings.NewReader("{invalid json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// testauthlogin_invalidbody verifies that a malformed request body returns 400.
func TestAuthLogin_InvalidBody(t *testing.T) {
	authHandler := handler.NewAuthHandler(nil)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/login", authHandler.Login)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// testjobcreate_invalidbody verifies that malformed json returns 400 before
// touching the service layer.
func TestJobCreate_InvalidBody(t *testing.T) {
	jobHandler := handler.NewJobHandler(nil)

	r := chi.NewRouter()
	r.Post("/api/v1/jobs", jobHandler.Create)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/jobs",
		strings.NewReader("{{not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid JSON, got %d", w.Code)
	}
}
