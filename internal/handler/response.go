package handler

import (
	"encoding/json"
	"net/http"

	"control-plane/internal/domain"
)

type ErrorResponse struct {
	Status  int    `json:"status"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func RespondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(payload)
}

func RespondError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	code := "internal_error"
	message := err.Error()

	switch {
	case err == domain.ErrNotFound:
		status = http.StatusNotFound
		code = "not_found"
	case err == domain.ErrConflict:
		status = http.StatusConflict
		code = "conflict"
	case err == domain.ErrInvalidInput:
		status = http.StatusBadRequest
		code = "invalid_input"
	case err == domain.ErrRollback:
		status = http.StatusUnprocessableEntity
		code = "rollback_failed"
	}

	RespondJSON(w, status, ErrorResponse{
		Status:  status,
		Code:    code,
		Message: message,
	})
}
