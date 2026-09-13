package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/myselfBZ/reserv-service/internal/store"
)

type healthCheckResponse struct {
	Status     string            `json:"status"`
	DeployedAt time.Time         `json:"deployed_at"`
	DB         *store.HealthInfo `json:"db"`
}

// HealthCheck godoc
//
//	@Summary		Health check
//	@Description	Reports service uptime and database health. Admin-only.
//	@Tags			health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	healthCheckResponse
//	@Success		503	{object}	healthCheckResponse
//	@Failure		401	{object}	ErrorResponse
//	@Failure		500	{object}	ErrorResponse
//	@Security		ApiKeyAuth
//	@Router			/health [get]
func (a *api) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	user := getUserFromContext(r)
	allowed, err := a.checkRolePrecedence(r.Context(), user, "admin")

	if err != nil {
		a.internalServerError(w, r, err)
		return
	}

	if !allowed {
		a.unauthorizedErrorResponse(w, r, fmt.Errorf("unauthorized attemt to access health"))
		return
	}

	h := &healthCheckResponse{
		Status:     "up",
		DeployedAt: a.startedAt,
		DB:         &store.HealthInfo{},
	}
	
	info, err := a.store.Health.Get()
	if err != nil {
		a.logger.Errorf("health check on database failed", "err", err)
		h.DB.Status = "down"
		writeJSON(w, http.StatusServiceUnavailable, h)
		return
	}
	h.DB = info
	writeJSON(w, http.StatusOK, h)
}
