package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/vishalyadav/jobplatform/internal/middleware"
	"github.com/vishalyadav/jobplatform/internal/models"
	"github.com/vishalyadav/jobplatform/internal/service"
	"github.com/vishalyadav/jobplatform/pkg/response"
)

type JobHandler struct {
	JobOrchestrator *service.JobOrchestrator
}

func NewJobHandler(JobOrchestrator *service.JobOrchestrator) *JobHandler {
	return &JobHandler{JobOrchestrator: JobOrchestrator}
}

func (h *JobHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req service.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")

	result, err := h.JobOrchestrator.CreateJob(r.Context(), userID, req, idempotencyKey)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusAccepted, result)
}

func (h *JobHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "jobID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	job, err := h.JobOrchestrator.GetJob(r.Context(), userID, jobID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, job)
}

func (h *JobHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	filters := models.JobListFilters{
		Page:  queryInt(r, "page", 1),
		Limit: queryInt(r, "limit", 20),
	}

	if s := r.URL.Query().Get("status"); s != "" {
		filters.Status = &s
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filters.Type = &t
	}
	if p := r.URL.Query().Get("priority"); p != "" {
		if pInt, err := strconv.Atoi(p); err == nil {
			filters.Priority = &pInt
		}
	}
	if ca := r.URL.Query().Get("created_after"); ca != "" {
		if t, err := time.Parse(time.RFC3339, ca); err == nil {
			filters.CreatedAfter = &t
		}
	}
	if cb := r.URL.Query().Get("created_before"); cb != "" {
		if t, err := time.Parse(time.RFC3339, cb); err == nil {
			filters.CreatedBefore = &t
		}
	}

	result, err := h.JobOrchestrator.ListJobs(r.Context(), userID, filters)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

func (h *JobHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "jobID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	if err := h.JobOrchestrator.CancelJob(r.Context(), userID, jobID); err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{
		"status": "cancelled",
	})
}

func (h *JobHandler) Logs(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	jobID, err := uuid.Parse(chi.URLParam(r, "jobID"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid job ID")
		return
	}

	logs, err := h.JobOrchestrator.GetJobLogs(r.Context(), userID, jobID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"logs": logs,
	})
}

func (h *JobHandler) Stats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	stats, err := h.JobOrchestrator.GetStats(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, stats)
}

// queryint parses an integer query parameter with a default value.
func queryInt(r *http.Request, key string, defaultVal int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return defaultVal
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return i
}
