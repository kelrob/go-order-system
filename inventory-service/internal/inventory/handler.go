package inventory

import (
	"net/http"

	"github.com/kelrob/shared/logger"
	"github.com/kelrob/shared/response"
)

type Handler struct {
	appLog *logger.Logger
}

func NewHandler(appLog *logger.Logger) *Handler {
	return &Handler{appLog: appLog}
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response.Success(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "inventory-service",
	})
}
