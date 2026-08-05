package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/response"
	"github.com/kelrob/shared/validation"
)

type AuthHandler struct {
	service *AuthService
}

func NewAuthHandler(service *AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response.Success(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "auth-service",
	})
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req SignupRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload", nil)
		return
	}

	if err := validation.ValidateRequest(req); err != nil {
		valErrs := err.(*validation.ValidationErrorCollection)
		response.Error(w, http.StatusBadRequest, "validation failed", valErrs.Errors)
		return
	}

	traceId := logger.TraceIdFromContext(r.Context())

	input := CreateUserInput{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		TraceId:   traceId,
	}

	user, err := h.service.CreateUser(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			response.Error(w, http.StatusConflict, err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong, please try again", nil)
		// TODO: Log err
		return
	}

	response.Success(w, http.StatusCreated, user)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request payload", nil)
		return
	}

	if err := validation.ValidateRequest(req); err != nil {
		valErrs := err.(*validation.ValidationErrorCollection)
		response.Error(w, http.StatusBadRequest, "validation failed", valErrs.Errors)
		return
	}

	traceId := logger.TraceIdFromContext(r.Context())

	input := LoginInput{
		Email:    req.Email,
		Password: req.Password,
		TraceId:  traceId,
	}

	authToken, err := h.service.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			response.Error(w, http.StatusUnauthorized, err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong, please try again", nil)
		// TODO: Log err
		return
	}

	response.Success(w, http.StatusOK, authToken)
}

func (h *AuthHandler) RefreshAccessToken(w http.ResponseWriter, r *http.Request) {
	var req RefreshTokenRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request payload", nil)
		return
	}

	if err := validation.ValidateRequest(req); err != nil {
		valErrs := err.(*validation.ValidationErrorCollection)
		response.Error(w, http.StatusBadRequest, "validation failed", valErrs.Errors)
		return
	}

	authToken, err := h.service.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrExpiredToken) {
			response.Error(w, http.StatusUnauthorized, err.Error(), nil)
			return
		}
		response.Error(w, http.StatusInternalServerError, "something went wrong, please try again", nil)
		// TODO: Log err
		return
	}

	response.Success(w, http.StatusOK, authToken)
}
