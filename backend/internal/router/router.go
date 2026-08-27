// package router sets up the chi router with all routes and middleware.
package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/vishalyadav/jobplatform/internal/config"
	"github.com/vishalyadav/jobplatform/internal/handler"
	"github.com/vishalyadav/jobplatform/internal/middleware"
	"github.com/vishalyadav/jobplatform/internal/service"
)

// new creates and configures the chi router with all routes.
func New(
	logger *slog.Logger,
	Authenticator *service.Authenticator,
	authHandler *handler.AuthHandler,
	jobHandler *handler.JobHandler,
	healthHandler *handler.HealthHandler,
	rdb *redis.Client,
	redisCfg config.RedisConfig,
) *chi.Mux {
	r := chi.NewRouter()

	// global middleware stack — order matters.
	r.Use(middleware.RequestID)      // inject request id first
	r.Use(middleware.Logger(logger)) // log every request
	r.Use(middleware.Metrics)        // prometheus metrics
	r.Use(chimw.Recoverer)           // catch panics → 500
	r.Use(chimw.RealIP)              // trust x-forwarded-for

	// cors for frontend development.
	r.Use(corsMiddleware)

	// rate limiting — applied globally, keyed per-user when authenticated.
	r.Use(middleware.RateLimit(rdb, middleware.RateLimitConfig{
		Requests: redisCfg.RateLimitRPS,
		Window:   time.Duration(redisCfg.RateLimitWindow) * time.Second,
	}, logger))

	// --- public routes ---

	// health endpoints (no auth needed — used by k8s probes).
	r.Get("/health/live", healthHandler.Live)
	r.Get("/health/ready", healthHandler.Ready)

	// prometheus metrics endpoint.
	r.Handle("/metrics", promhttp.Handler())

	// auth endpoints.
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// --- protected routes ---
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(Authenticator))

		r.Route("/api/v1/jobs", func(r chi.Router) {
			r.Post("/", jobHandler.Create)
			r.Get("/", jobHandler.List)
			r.Get("/stats", jobHandler.Stats)
			r.Get("/{jobID}", jobHandler.Get)
			r.Post("/{jobID}/cancel", jobHandler.Cancel)
			r.Get("/{jobID}/logs", jobHandler.Logs)
		})
	})

	return r
}

// corsmiddleware adds permissive cors headers for development.
// in production, this should be restricted to specific origins.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
