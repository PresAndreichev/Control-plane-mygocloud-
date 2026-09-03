package handler

import (
	"net/http"

	"control-plane/internal/executor"

	"github.com/go-chi/chi/v5"
)

type InfraHandler struct {
	exec executor.Executor
}

func NewInfraHandler(exec executor.Executor) *InfraHandler {
	return &InfraHandler{exec: exec}
}

func (h *InfraHandler) Routes(r chi.Router) {
	r.Get("/infra/containers", h.ListContainers)
}

func (h *InfraHandler) ListContainers(w http.ResponseWriter, r *http.Request) {
	// This would need the executor to support ListContainers
	// For now, skip or return placeholder
	RespondJSON(w, http.StatusOK, map[string]string{"status": "infra endpoint ready"})
}
