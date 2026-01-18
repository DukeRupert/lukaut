package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "lukaut"

// HTTP metrics
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "http_requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request latency distribution",
			Buckets:   []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "http_requests_in_flight",
			Help:      "Current number of HTTP requests being processed",
		},
	)
)

// Background job metrics
var (
	JobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "jobs_total",
			Help:      "Total number of jobs processed",
		},
		[]string{"type", "status"},
	)

	JobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "job_duration_seconds",
			Help:      "Job execution time distribution",
			Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
		[]string{"type"},
	)

	JobRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "job_retries_total",
			Help:      "Total number of job retry attempts",
		},
		[]string{"type"},
	)
)

// Business metrics
var (
	InspectionsCreated = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "inspections_created_total",
			Help:      "Total number of inspections created",
		},
	)

	ReportsGenerated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "reports_generated_total",
			Help:      "Total number of reports generated",
		},
		[]string{"format"},
	)

	AIAPICalls = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ai_api_calls_total",
			Help:      "Total number of AI API calls",
		},
		[]string{"status"},
	)

	ImagesAnalyzed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "images_analyzed_total",
			Help:      "Total number of images analyzed",
		},
		[]string{"status"},
	)

	ViolationsDetected = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "violations_detected_total",
			Help:      "Total number of violations detected by AI",
		},
	)
)

// AI cost tracking metrics (aggregate totals - no user label to avoid cardinality)
var (
	AITokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ai_tokens_total",
			Help:      "Total AI tokens consumed",
		},
		[]string{"type"}, // "input" or "output"
	)

	AICostCentsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "ai_cost_cents_total",
			Help:      "Total AI cost in cents",
		},
	)
)
