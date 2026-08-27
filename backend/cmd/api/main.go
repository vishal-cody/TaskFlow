// api server entrypoint.
//
// wires up all dependencies and starts the http server with graceful shutdown.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vishalyadav/jobplatform/internal/config"
	"github.com/vishalyadav/jobplatform/internal/database"
	"github.com/vishalyadav/jobplatform/internal/handler"
	"github.com/vishalyadav/jobplatform/internal/queue"
	redisclient "github.com/vishalyadav/jobplatform/internal/redis"
	"github.com/vishalyadav/jobplatform/internal/repository"
	"github.com/vishalyadav/jobplatform/internal/router"
	"github.com/vishalyadav/jobplatform/internal/service"
)

func main() {
	// ── logger ──
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// ── config ──
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// set log level from config.
	var logLevel slog.Level
	switch cfg.Log.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// ── database ──
	ctx := context.Background()
	pool, err := database.NewPostgresPool(ctx, cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	logger.Info("connected to PostgreSQL")

	// ── rabbitmq ──
	amqpConn, err := queue.NewConnection(ctx, cfg.RabbitMQ.URL, logger)
	if err != nil {
		logger.Error("failed to connect to RabbitMQ", "error", err)
		os.Exit(1)
	}
	defer amqpConn.Close()

	// ── redis ──
	rdb, err := redisclient.NewClient(ctx, cfg.Redis.URL, logger)
	if err != nil {
		logger.Error("failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// ── repositories ──
	userRepo := repository.NewUserRepository(pool)
	jobRepo := repository.NewJobRepository(pool)
	jobLogRepo := repository.NewJobLogRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)

	// ── services ──
	Authenticator := service.NewAuthenticator(userRepo, cfg.JWT)
	JobOrchestrator := service.NewJobOrchestrator(jobRepo, jobLogRepo, outboxRepo, logger)

	// ── background tasks ──
	outboxPublisher := queue.NewOutboxPublisher(outboxRepo, amqpConn, logger, 2*time.Second)
	outboxPublisher.Start(ctx)

	// ── handlers ──
	authHandler := handler.NewAuthHandler(Authenticator)
	jobHandler := handler.NewJobHandler(JobOrchestrator)
	healthHandler := handler.NewHealthHandler(pool, rdb)

	// ── router ──
	r := router.New(logger, Authenticator, authHandler, jobHandler, healthHandler, rdb, cfg.Redis)

	// ── http server ──
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// ── graceful shutdown ──
	//
	// listen for sigint/sigterm, then:
	//  1. stop accepting new connections
	//  2. wait for in-flight requests to finish (up to shutdowntimeout)
	//  3. close db pool
	//  4. exit
	done := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("received shutdown signal", "signal", sig.String())

		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown error", "error", err)
		}
		
		outboxPublisher.Stop()
		close(done)
	}()

	logger.Info("starting API server", "addr", addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	<-done
	logger.Info("server stopped gracefully")
}
