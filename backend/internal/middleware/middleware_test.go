package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID_Injected(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id == "" {
			t.Error("expected non-empty request ID in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// verify the response header was set.
	rid := w.Header().Get("X-Request-ID")
	if rid == "" {
		t.Error("expected X-Request-ID response header to be set")
	}
}

func TestRequestID_PreservesClientHeader(t *testing.T) {
	clientID := "client-provided-id-123"

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := RequestIDFromContext(r.Context())
		if id != clientID {
			t.Errorf("expected request ID %q, got %q", clientID, id)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", clientID)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	rid := w.Header().Get("X-Request-ID")
	if rid != clientID {
		t.Errorf("expected X-Request-ID %q, got %q", clientID, rid)
	}
}
