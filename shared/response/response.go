package response

import (
	"encoding/json"
	"net/http"
)

type SuccessResponse[T any] struct {
	Success bool `json:"success"`
	Status  int  `json:"status"`
	Data    T    `json:"data"`
}

type ErrorResponse struct {
	Success bool   `json:"success"`
	Status  int    `json:"status"`
	Error   string `json:"error"`
	Details any    `json:"details,omitempty"`
}

func Success[T any](w http.ResponseWriter, status int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(SuccessResponse[T]{
		Success: true,
		Status:  status,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, message string, details any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{
		Success: false,
		Status:  status,
		Error:   message,
		Details: details,
	})
}
