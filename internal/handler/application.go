package handler

import (
	"control-plane/internal/domain"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type ApplicationHandler struct {
	appService domain.ApplicationService
	depService domain.DeploymentService
}

func NewApplicationHandler(appService domain.ApplicationService, depService domain.DeploymentService) *ApplicationHandler {
	return &ApplicationHandler{
		appService: appService,
		depService: depService,
	}
}

func (h *ApplicationHandler) Routes(r chi.Router) {
	r.Post("/applications", h.Create)
	r.Get("/applications", h.List)
	r.Get("/applications/{id}", h.Get)
	r.Post("/applications/{id}/deploy", h.Deploy)
	r.Get("/applications/{id}/deployments", h.ListDeployments)
	r.Post("/applications/{id}/rollback", h.Rollback)
	r.Get("/deployments/logs", h.GetLogs)

}

func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		DockerImage string    `json:"docker_image"`
		OwnerID     uuid.UUID `json:"owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	app, err := h.appService.Create(r.Context(), req.Name, req.Description, req.DockerImage, req.OwnerID)
	if err != nil {
		// TEMPORARY DEBUG: log the actual error
		slog.Error("failed to create application", "error", err, "owner_id", req.OwnerID)
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusCreated, app)
}

func (h *ApplicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	app, err := h.appService.GetByID(r.Context(), id)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, app)
}

func (h *ApplicationHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	var req struct {
		Version string `json:"version"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	dep, err := h.depService.Deploy(r.Context(), appID, req.Version)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusAccepted, dep)
}

func (h *ApplicationHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	deps, err := h.depService.ListByApplication(r.Context(), appID)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, deps)
}

func (h *ApplicationHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	dep, err := h.depService.Rollback(r.Context(), appID)
	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusAccepted, dep)
}

func (h *ApplicationHandler) List(w http.ResponseWriter, r *http.Request) {
	ownerIDStr := r.URL.Query().Get("owner_id")

	var apps []domain.Application
	var err error

	if ownerIDStr != "" {
		ownerID, parseErr := uuid.Parse(ownerIDStr)
		if parseErr != nil {
			RespondError(w, domain.ErrInvalidInput)
			return
		}
		apps, err = h.appService.ListByOwner(r.Context(), ownerID)
	} else {
		apps, err = h.appService.List(r.Context())
	}

	if err != nil {
		RespondError(w, err)
		return
	}
	RespondJSON(w, http.StatusOK, apps)
}

func (h *ApplicationHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.URL.Query().Get("container_id")
	tailStr := r.URL.Query().Get("tail")

	if containerID == "" {
		RespondError(w, domain.ErrInvalidInput)
		return
	}

	tail := 100
	if t, err := strconv.Atoi(tailStr); err == nil && t > 0 {
		tail = t
	}

	logs, err := h.depService.GetLogs(r.Context(), containerID, tail)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{"logs": logs})
}
