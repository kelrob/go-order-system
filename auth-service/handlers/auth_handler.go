package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kelrob/auth-service/domain"
	"github.com/kelrob/auth-service/service"
	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/response"
	"github.com/kelrob/shared/validation"
)

type AuthHandler struct {
	service *service.AuthService
}

type CreateUserRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}
func (h *AuthHandler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error(), nil)
		return
	}

	if err := validation.ValidateRequest(req); err != nil {
		valErrs := err.(*validation.ValidationErrorCollection)
		response.Error(w, http.StatusBadRequest, "validation failed", valErrs.Errors)
		return
	}

	traceId := logger.TraceIdFromContext(r.Context())

	input := service.CreateUserInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		TraceId:   traceId,
	}

	user, err := h.service.CreateUser(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrEmailAlreadyExists) {
			response.Error(w, http.StatusConflict, err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong, please try again", nil)
		// TODO: Log err
		return
	}

	response.Success(w, http.StatusCreated, user)
}
