// package metrics defines application-wide prometheus metrics.
//
// all metrics follow the pattern:
//
//	jobplatform_<subsystem>_<name>_{total,seconds,etc}
//
// this package is imported by middleware, services, and the worker to
// record http, queue, and job-processing telemetry.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "jobplatform"

// ─── http metrics ────────────────────────────────────────────────────────────

// httprequeststotal counts total http requests by method, path, and status.
var HTTPRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "requests_total",
		Help:      "Total number of HTTP requests.",
	},
	[]string{"method", "path", "status"},
)

// httprequestduration tracks request latency in seconds by method and path.
var HTTPRequestDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "HTTP request duration in seconds.",
		Buckets:   prometheus.DefBuckets,
	},
	[]string{"method", "path"},
)

// httprequestsinflight tracks the number of concurrent http requests.
var HTTPRequestsInFlight = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "http",
		Name:      "requests_in_flight",
		Help:      "Number of HTTP requests currently being served.",
	},
)

// ─── job metrics ─────────────────────────────────────────────────────────────

// jobscreatedtotal counts jobs created by type.
var JobsCreatedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "jobs",
		Name:      "created_total",
		Help:      "Total number of jobs created.",
	},
	[]string{"type"},
)

// jobsprocessedtotal counts jobs processed by type and final status.
var JobsProcessedTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "jobs",
		Name:      "processed_total",
		Help:      "Total number of jobs processed.",
	},
	[]string{"type", "status"},
)

// jobprocessingduration tracks how long a job takes to process end-to-end.
var JobProcessingDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: "jobs",
		Name:      "processing_duration_seconds",
		Help:      "Job processing duration in seconds.",
		Buckets:   []float64{0.1, 0.5, 1, 5, 10, 30, 60, 120, 300},
	},
	[]string{"type"},
)

// jobsinflight tracks the number of jobs currently being processed.
var JobsInFlight = promauto.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "jobs",
		Name:      "in_flight",
		Help:      "Number of jobs currently being processed by workers.",
	},
)

// ─── queue metrics ───────────────────────────────────────────────────────────

// outboxpublishedtotal counts events published from the outbox to rabbitmq.
var OutboxPublishedTotal = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "outbox",
		Name:      "published_total",
		Help:      "Total number of outbox events published to RabbitMQ.",
	},
)

// outboxpublisherrors counts failures when publishing from the outbox.
var OutboxPublishErrors = promauto.NewCounter(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: "outbox",
		Name:      "publish_errors_total",
		Help:      "Total number of outbox publish failures.",
	},
)
