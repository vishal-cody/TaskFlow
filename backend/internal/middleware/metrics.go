package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/vishalyadav/jobplatform/internal/metrics"
)

// responsewriter wraps http.responsewriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// metrics returns middleware that records prometheus metrics for every request.
//
// it tracks:
//   - jobplatform_http_requests_total        (counter, by method/path/status)
//   - jobplatform_http_request_duration_seconds (histogram, by method/path)
//   - jobplatform_http_requests_in_flight    (gauge)
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// track in-flight requests.
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// wrap the response writer to capture the status code.
		wrapped := newResponseWriter(w)
		next.ServeHTTP(wrapped, r)

		duration := time.Since(start).Seconds()

		// use the chi route pattern (e.g. "/api/v1/jobs/{jobid}") instead of
		// the actual url to avoid high-cardinality label explosion.
		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "unknown"
		}

		status := strconv.Itoa(wrapped.statusCode)

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, routePattern, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, routePattern).Observe(duration)
	})
}
