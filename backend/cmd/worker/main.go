// worker binary entrypoint.
//
// connects to postgresql and rabbitmq, registers job processors,
// and starts the consumer loop.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vishalyadav/jobplatform/internal/config"
	"github.com/vishalyadav/jobplatform/internal/database"
	"github.com/vishalyadav/jobplatform/internal/models"
	"github.com/vishalyadav/jobplatform/internal/queue"
	"github.com/vishalyadav/jobplatform/internal/repository"
	"github.com/vishalyadav/jobplatform/internal/worker"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── database ──
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

	// ── repositories ──
	jobRepo := repository.NewJobRepository(pool)
	jobLogRepo := repository.NewJobLogRepository(pool)

	// ── processors ──
	registry := worker.NewProcessorRegistry()
	registry.Register(models.JobTypeReportGeneration, &worker.ReportGenerationProcessor{})
	registry.Register(models.JobTypeDataProcessing, &worker.DataProcessingProcessor{})
	// jobtypeimageprocessing and jobtypenotification are valid models, but let's 
	// leave them unimplemented for now to see what happens when a worker gets an unregistered type.
	// (it should fail the job).

	// ── consumer ──
	consumer := worker.NewConsumer(
		amqpConn,
		jobRepo,
		jobLogRepo,
		registry,
		logger,
		5, // concurrency limit (prefetch count)
	)

	// ── metrics server ──
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		logger.Info("starting worker metrics server", "addr", ":8081")
		if err := http.ListenAndServe(":8081", mux); err != nil {
			logger.Error("metrics server failed", "error", err)
		}
	}()

	// ── graceful shutdown ──
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigCh
		logger.Info("received shutdown signal, stopping consumer", "signal", sig.String())
		cancel() // this unblocks consumer.start()
	}()

	// blocks until ctx is cancelled and active jobs finish.
	if err := consumer.Start(ctx); err != nil {
		logger.Error("consumer stopped with error", "error", err)
		os.Exit(1)
	}

	logger.Info("worker stopped cleanly")
}
