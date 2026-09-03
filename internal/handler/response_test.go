package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"control-plane/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestRespondJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondJSON(rec, http.StatusOK, map[string]string{"key": "value"})

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]string
	err := json.NewDecoder(rec.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, "value", resp["key"])
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
		wantCode   string
	}{
		{domain.ErrNotFound, http.StatusNotFound, "not_found"},
		{domain.ErrConflict, http.StatusConflict, "conflict"},
		{domain.ErrInvalidInput, http.StatusBadRequest, "invalid_input"},
		{domain.ErrRollback, http.StatusUnprocessableEntity, "rollback_failed"},
		{assert.AnError, http.StatusInternalServerError, "internal_error"},
	}

	for _, tt := range tests {
		rec := httptest.NewRecorder()
		RespondError(rec, tt.err)

		assert.Equal(t, tt.wantStatus, rec.Code)

		var resp ErrorResponse
		json.NewDecoder(rec.Body).Decode(&resp)
		assert.Equal(t, tt.wantCode, resp.Code)
		assert.Equal(t, tt.wantStatus, resp.Status)
	}
}
