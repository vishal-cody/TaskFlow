package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vishalyadav/jobplatform/internal/service"
	"github.com/vishalyadav/jobplatform/internal/validator"
	"github.com/vishalyadav/jobplatform/pkg/response"
)

type AuthHandler struct {
	Authenticator *service.Authenticator
}

func NewAuthHandler(Authenticator *service.Authenticator) *AuthHandler {
	return &AuthHandler{Authenticator: Authenticator}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req service.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.Authenticator.Register(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req service.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.Authenticator.Login(r.Context(), req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// handleserviceerror maps typed service errors to http status codes.
// this is the single place where business errors become http responses.
func handleServiceError(w http.ResponseWriter, err error) {
	var conflictErr *service.ConflictError
	var unauthorizedErr *service.UnauthorizedError
	var notFoundErr *service.NotFoundError
	var forbiddenErr *service.ForbiddenError
	var badRequestErr *service.BadRequestError
	var validationErr validator.Errors

	switch {
	case errors.As(err, &validationErr):
		response.JSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"error": map[string]interface{}{
				"message": "validation failed",
				"fields":  validationErr,
			},
		})
	case errors.As(err, &conflictErr):
		response.Error(w, http.StatusConflict, conflictErr.Message)
	case errors.As(err, &unauthorizedErr):
		response.Error(w, http.StatusUnauthorized, unauthorizedErr.Message)
	case errors.As(err, &notFoundErr):
		response.Error(w, http.StatusNotFound, notFoundErr.Message)
	case errors.As(err, &forbiddenErr):
		response.Error(w, http.StatusForbidden, forbiddenErr.Message)
	case errors.As(err, &badRequestErr):
		response.Error(w, http.StatusBadRequest, badRequestErr.Message)
	default:
		// don't leak internal errors to clients.
		response.Error(w, http.StatusInternalServerError, "internal server error")
	}
}
