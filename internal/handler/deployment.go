package handler

import (
	"net/http"
	"strconv"

	"control-plane/internal/domain"

	"github.com/go-chi/chi/v5"
)

type DeploymentHandler struct {
	depService domain.DeploymentService
}

func NewDeploymentHandler(depService domain.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{depService: depService}
}

func (h *DeploymentHandler) Routes(r chi.Router) {
	r.Get("/deployments/logs", h.GetLogs)
}

func (h *DeploymentHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("container_id")
	if containerID == "" {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	tail := 100
	if t, err := strconv.Atoi(r.URL.Query().Get("tail")); err == nil && t > 0 {
		tail = t
	}

	logs, err := h.depService.GetLogs(r.Context(), containerID, tail)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"logs": logs})
}
