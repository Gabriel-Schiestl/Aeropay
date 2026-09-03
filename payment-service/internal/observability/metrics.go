package observability

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests, by method, path and status",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration, by method, path and status",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	// paymentProcessingDuration measures real acceptance-to-settlement
	// latency (time.Since(event.AcceptedAt) at the moment the worker
	// finishes processing), not just the worker's own local processing
	// time - it includes time spent waiting in the outbox and in Kafka.
	paymentProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_processing_duration_seconds",
			Help:    "Time from payment acceptance (202) to settlement (processed by a worker), by outcome",
			Buckets: []float64{.05, .1, .25, .5, .75, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"outcome"},
	)

	// paymentsOutboxPending tracks whether the outbox backlog is draining
	// or growing - the single most direct signal of whether processing is
	// keeping up with acceptance under load.
	paymentsOutboxPending = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "payments_outbox_pending",
			Help: "Number of payments_outbox rows currently in status='pending'",
		},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal, httpRequestDuration, paymentProcessingDuration, paymentsOutboxPending)
}

// RecordPaymentProcessing records the acceptance-to-settlement latency for
// one payment. outcome should be "success" or "error".
func RecordPaymentProcessing(acceptedAt time.Time, outcome string) {
	paymentProcessingDuration.WithLabelValues(outcome).Observe(time.Since(acceptedAt).Seconds())
}

// SetOutboxPending reports the current size of the pending outbox backlog.
func SetOutboxPending(count float64) {
	paymentsOutboxPending.Set(count)
}

func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())

		httpRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		httpRequestDuration.WithLabelValues(c.Request.Method, path, status).Observe(time.Since(start).Seconds())
	}
}

func RegisterDBCollector(db *sql.DB) {
	prometheus.MustRegister(collectors.NewDBStatsCollector(db, "payment_service"))
}

func Handler() gin.HandlerFunc {
	return gin.WrapH(promhttp.Handler())
}
