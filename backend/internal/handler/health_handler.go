package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/vishalyadav/jobplatform/pkg/response"
)

type HealthHandler struct {
	pool *pgxpool.Pool
	rdb  *redis.Client
}

func NewHealthHandler(pool *pgxpool.Pool, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{pool: pool, rdb: rdb}
}

// kubernetes uses this as the liveness probe.
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

// kubernetes uses this as the readiness probe.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	deps := map[string]string{}
	ready := true

	// check postgresql
	if err := h.pool.Ping(r.Context()); err != nil {
		deps["postgres"] = "unavailable"
		ready = false
	} else {
		deps["postgres"] = "ok"
	}

	// check redis
	if err := h.rdb.Ping(r.Context()).Err(); err != nil {
		deps["redis"] = "unavailable"
		ready = false
	} else {
		deps["redis"] = "ok"
	}

	status := "ready"
	httpStatus := http.StatusOK
	if !ready {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	response.JSON(w, httpStatus, map[string]interface{}{
		"status":       status,
		"dependencies": deps,
	})
}
