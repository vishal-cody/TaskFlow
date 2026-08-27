package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/vishalyadav/jobplatform/internal/service"
	"github.com/vishalyadav/jobplatform/pkg/response"
)

// contextkey is an unexported type for context keys to avoid collisions.
type contextKey string

const userIDKey contextKey = "user_id"

// auth returns middleware that validates jwt bearer tokens and injects
// the authenticated user id into the request context.
func Auth(Authenticator *service.Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				response.Error(w, http.StatusUnauthorized, "missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				response.Error(w, http.StatusUnauthorized, "invalid authorization format")
				return
			}

			claims, err := Authenticator.ValidateToken(parts[1])
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// useridfromcontext extracts the authenticated user id from context.
// returns uuid.nil if not present (should never happen behind auth middleware).
func UserIDFromContext(ctx context.Context) uuid.UUID {
	id, ok := ctx.Value(userIDKey).(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	return id
}
