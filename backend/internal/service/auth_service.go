package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/vishalyadav/jobplatform/internal/config"
	"github.com/vishalyadav/jobplatform/internal/models"
	"github.com/vishalyadav/jobplatform/internal/repository"
	"github.com/vishalyadav/jobplatform/internal/validator"
	"golang.org/x/crypto/bcrypt"
)

type Authenticator struct {
	userRepo *repository.UserRepository
	jwtCfg   config.JWTConfig
}

func NewAuthenticator(userRepo *repository.UserRepository, jwtCfg config.JWTConfig) *Authenticator {
	return &Authenticator{
		userRepo: userRepo,
		jwtCfg:   jwtCfg,
	}
}

// registerrequest holds registration input.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// registerresponse holds registration output.
type RegisterResponse struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

// register creates a new user after validating input and hashing the password.
func (s *Authenticator) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	// validate input.
	errs := validator.Errors{}
	if err := validator.ValidateEmail(req.Email); err != nil {
		errs["email"] = err.Error()
	}
	if err := validator.ValidatePassword(req.Password); err != nil {
		errs["password"] = err.Error()
	}
	if errs.HasErrors() {
		return nil, errs
	}

	// check for existing user.
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}
	if existing != nil {
		return nil, &ConflictError{Message: "email already registered"}
	}

	// hash password.
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return &RegisterResponse{
		ID:    user.ID,
		Email: user.Email,
	}, nil
}

// loginrequest holds login input.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginresponse holds login output.
type LoginResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// login validates credentials and issues a jwt.
func (s *Authenticator) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	if req.Email == "" || req.Password == "" {
		return nil, &UnauthorizedError{Message: "email and password are required"}
	}

	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}
	if user == nil {
		return nil, &UnauthorizedError{Message: "invalid credentials"}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, &UnauthorizedError{Message: "invalid credentials"}
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generating token: %w", err)
	}

	return &LoginResponse{
		AccessToken: token,
		ExpiresIn:   int(s.jwtCfg.Expiry.Seconds()),
	}, nil
}

// claims represents jwt claims.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	jwt.RegisteredClaims
}

// generatetoken creates a signed jwt for the given user.
func (s *Authenticator) generateToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtCfg.Expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtCfg.Secret))
}

// validatetoken parses and validates a jwt string. returns the claims if valid.
func (s *Authenticator) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtCfg.Secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// --- typed service errors ---

// conflicterror indicates a resource already exists.
type ConflictError struct{ Message string }

func (e *ConflictError) Error() string { return e.Message }

// unauthorizederror indicates invalid credentials.
type UnauthorizedError struct{ Message string }

func (e *UnauthorizedError) Error() string { return e.Message }

// notfounderror indicates a resource was not found.
type NotFoundError struct{ Message string }

func (e *NotFoundError) Error() string { return e.Message }

// forbiddenerror indicates the user lacks permission.
type ForbiddenError struct{ Message string }

func (e *ForbiddenError) Error() string { return e.Message }

// badrequesterror indicates invalid input.
type BadRequestError struct{ Message string }

func (e *BadRequestError) Error() string { return e.Message }
