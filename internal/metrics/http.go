package metrics

import (
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// uuidPattern matches UUIDs in paths for normalization
var uuidPattern = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

// responseWriter wraps http.ResponseWriter to capture status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
	wroteHeader  bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += n
	return n, err
}

// Unwrap returns the underlying ResponseWriter for middleware compatibility
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// normalizePath replaces UUIDs with {id} to reduce metric cardinality
func normalizePath(path string) string {
	return uuidPattern.ReplaceAllString(path, "{id}")
}

// Middleware records HTTP request metrics
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip metrics endpoint to avoid recursion
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		HTTPRequestsInFlight.Inc()
		defer HTTPRequestsInFlight.Dec()

		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()
		path := normalizePath(r.URL.Path)
		method := r.Method
		statusCode := strconv.Itoa(rw.statusCode)

		HTTPRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
		HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
	})
}
