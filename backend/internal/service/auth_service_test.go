package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vishalyadav/jobplatform/internal/config"
)

func TestAuthenticator_GenerateAndValidateToken(t *testing.T) {
	cfg := config.JWTConfig{
		Secret: "test-secret-key-for-unit-tests",
		Expiry: 1 * time.Hour,
	}

	// we can't construct authenticator with newauthenticator because it needs
	// a real userrepository, but we only want to test token logic.
	// use the exported validatetoken + unexported generatetoken via the struct.
	svc := &Authenticator{
		userRepo: nil, // not needed for token tests
		jwtCfg:   cfg,
	}

	userID := uuid.New()

	t.Run("valid token round-trip", func(t *testing.T) {
		token, err := svc.generateToken(userID)
		if err != nil {
			t.Fatalf("generateToken() error = %v", err)
		}
		if token == "" {
			t.Fatal("generateToken() returned empty token")
		}

		claims, err := svc.ValidateToken(token)
		if err != nil {
			t.Fatalf("ValidateToken() error = %v", err)
		}
		if claims.UserID != userID {
			t.Errorf("expected userID %v, got %v", userID, claims.UserID)
		}
	})

	t.Run("different secrets reject token", func(t *testing.T) {
		token, _ := svc.generateToken(userID)

		otherSvc := &Authenticator{
			jwtCfg: config.JWTConfig{
				Secret: "different-secret",
				Expiry: 1 * time.Hour,
			},
		}

		_, err := otherSvc.ValidateToken(token)
		if err == nil {
			t.Error("expected error when validating token with different secret")
		}
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		expiredSvc := &Authenticator{
			jwtCfg: config.JWTConfig{
				Secret: cfg.Secret,
				Expiry: -1 * time.Hour, // already expired
			},
		}

		token, _ := expiredSvc.generateToken(userID)

		_, err := svc.ValidateToken(token)
		if err == nil {
			t.Error("expected error for expired token")
		}
	})

	t.Run("malformed token is rejected", func(t *testing.T) {
		_, err := svc.ValidateToken("not.a.valid.jwt")
		if err == nil {
			t.Error("expected error for malformed token")
		}
	})

	t.Run("empty token is rejected", func(t *testing.T) {
		_, err := svc.ValidateToken("")
		if err == nil {
			t.Error("expected error for empty token")
		}
	})
}
