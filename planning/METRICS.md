# Prometheus Metrics Guide

This document describes the Prometheus metrics exposed by Lukaut for monitoring and observability.

## Endpoint

```
GET /metrics
```

Public endpoint (no authentication required). Returns metrics in Prometheus text format.

## Metrics Reference

### HTTP Metrics

Collected automatically via middleware for all requests (except `/metrics` itself).

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `lukaut_http_requests_total` | Counter | `method`, `path`, `status_code` | Total HTTP requests |
| `lukaut_http_request_duration_seconds` | Histogram | `method`, `path` | Request latency (buckets: 5ms to 10s) |
| `lukaut_http_requests_in_flight` | Gauge | - | Currently processing requests |

**Path normalization:** UUIDs in paths are replaced with `{id}` to prevent high cardinality (e.g., `/inspections/abc-123` becomes `/inspections/{id}`).

### Background Job Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `lukaut_jobs_total` | Counter | `type`, `status` | Jobs processed (status: `completed`, `failed`) |
| `lukaut_job_duration_seconds` | Histogram | `type` | Job execution time (buckets: 1s to 10min) |
| `lukaut_job_retries_total` | Counter | `type` | Job retry attempts |

**Job types:** `analyze_inspection`, `generate_report`

### Business Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `lukaut_inspections_created_total` | Counter | - | Inspections created |
| `lukaut_reports_generated_total` | Counter | `format` | Reports generated (`pdf`, `docx`) |
| `lukaut_ai_api_calls_total` | Counter | `status` | AI API calls (`success`, `error`) |
| `lukaut_images_analyzed_total` | Counter | `status` | Images analyzed (`success`, `error`) |
| `lukaut_violations_detected_total` | Counter | - | Violations detected by AI |

### AI Cost Metrics

Aggregate totals (no user labels to avoid cardinality issues). Use SQL queries for per-customer breakdown.

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `lukaut_ai_tokens_total` | Counter | `type` | Total tokens consumed (`input`, `output`) |
| `lukaut_ai_cost_cents_total` | Counter | - | Total AI cost in cents |

## Example Queries

### Request rate (last 5 minutes)
```promql
rate(lukaut_http_requests_total[5m])
```

### 95th percentile latency by endpoint
```promql
histogram_quantile(0.95, rate(lukaut_http_request_duration_seconds_bucket[5m]))
```

### Error rate
```promql
sum(rate(lukaut_http_requests_total{status_code=~"5.."}[5m]))
/ sum(rate(lukaut_http_requests_total[5m]))
```

### Job success rate
```promql
sum(rate(lukaut_jobs_total{status="completed"}[1h]))
/ sum(rate(lukaut_jobs_total[1h]))
```

### AI API error rate
```promql
sum(rate(lukaut_ai_api_calls_total{status="error"}[1h]))
/ sum(rate(lukaut_ai_api_calls_total[1h]))
```

### Total AI cost (dollars)
```promql
lukaut_ai_cost_cents_total / 100
```

### AI cost rate (cents per hour)
```promql
rate(lukaut_ai_cost_cents_total[1h]) * 3600
```

### Token usage rate
```promql
sum(rate(lukaut_ai_tokens_total[1h])) by (type)
```

## Per-Customer AI Usage (SQL)

For per-customer cost breakdown, use the SQL queries in `sqlc/queries/ai_usage.sql`:

| Query | Description |
|-------|-------------|
| `GetUserAIUsageThisMonth` | Current month usage for a user |
| `GetUserAIUsageByDateRange` | Usage for a user in date range |
| `GetAllUsersAIUsageSummary` | All users usage summary (billing report) |
| `GetPlatformAIUsageTotal` | Total platform usage |
| `GetUserAIUsageByDay` | Daily breakdown for charts |

Example usage in Go:
```go
// Get all users' AI costs for January 2024
startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
endDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
summary, err := queries.GetAllUsersAIUsageSummary(ctx, startDate, endDate)
```

## Files

- `internal/metrics/metrics.go` - Metric definitions
- `internal/metrics/http.go` - HTTP middleware
- `internal/metrics/worker.go` - Job metric helpers
- `sqlc/queries/ai_usage.sql` - Per-customer reporting queries

## Adding New Metrics

1. Define the metric in `internal/metrics/metrics.go` using `promauto`
2. Import the metrics package where needed
3. Call the appropriate method (`.Inc()`, `.Add()`, `.Observe()`, `.Set()`)

Example:
```go
// In metrics.go
var MyCounter = promauto.NewCounter(prometheus.CounterOpts{
    Namespace: namespace,
    Name:      "my_counter_total",
    Help:      "Description of what this counts",
})

// In your code
metrics.MyCounter.Inc()
```
