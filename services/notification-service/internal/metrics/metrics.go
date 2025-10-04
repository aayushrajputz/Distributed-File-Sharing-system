package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/yourusername/distributed-file-sharing/services/notification-service/internal/models"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Notification metrics
	NotificationsSentTotal              *prometheus.CounterVec
	NotificationsDeliveryDuration       *prometheus.HistogramVec
	NotificationsRetryTotal             *prometheus.CounterVec
	NotificationsDLQTotal               *prometheus.CounterVec
	NotificationsBatchedTotal           prometheus.Counter
	NotificationPreferencesUpdatedTotal prometheus.Counter

	// Channel metrics
	ChannelConnectionsTotal *prometheus.GaugeVec
	ChannelErrorsTotal      *prometheus.CounterVec

	// Batch metrics
	BatchSizeHistogram      prometheus.Histogram
	BatchProcessingDuration prometheus.Histogram

	// DLQ metrics
	DLQEntriesTotal       prometheus.Gauge
	DLQRetryAttemptsTotal *prometheus.CounterVec

	// System metrics
	ActiveConnections     prometheus.Gauge
	ProcessingErrorsTotal *prometheus.CounterVec
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		// Notification metrics
		NotificationsSentTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_sent_total",
				Help: "Total number of notifications sent",
			},
			[]string{"channel", "event_type", "status"},
		),
		NotificationsDeliveryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "notifications_delivery_duration_seconds",
				Help:    "Time taken to deliver notifications",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"channel"},
		),
		NotificationsRetryTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_retry_total",
				Help: "Total number of notification retries",
			},
			[]string{"channel", "attempt"},
		),
		NotificationsDLQTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "notifications_dlq_total",
				Help: "Total number of notifications sent to DLQ",
			},
			[]string{"event_type"},
		),
		NotificationsBatchedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "notifications_batched_total",
				Help: "Total number of notifications batched",
			},
		),
		NotificationPreferencesUpdatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "notification_preferences_updated_total",
				Help: "Total number of preference updates",
			},
		),

		// Channel metrics
		ChannelConnectionsTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "channel_connections_total",
				Help: "Total number of active channel connections",
			},
			[]string{"channel"},
		),
		ChannelErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "channel_errors_total",
				Help: "Total number of channel errors",
			},
			[]string{"channel", "error_type"},
		),

		// Batch metrics
		BatchSizeHistogram: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "batch_size_histogram",
				Help:    "Distribution of batch sizes",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			},
		),
		BatchProcessingDuration: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "batch_processing_duration_seconds",
				Help:    "Time taken to process batches",
				Buckets: prometheus.DefBuckets,
			},
		),

		// DLQ metrics
		DLQEntriesTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "dlq_entries_total",
				Help: "Total number of entries in DLQ",
			},
		),
		DLQRetryAttemptsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "dlq_retry_attempts_total",
				Help: "Total number of DLQ retry attempts",
			},
			[]string{"event_type", "status"},
		),

		// System metrics
		ActiveConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections_total",
				Help: "Total number of active connections",
			},
		),
		ProcessingErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "processing_errors_total",
				Help: "Total number of processing errors",
			},
			[]string{"service", "error_type"},
		),
	}
}

// RecordNotificationSent records a notification sent
func (m *Metrics) RecordNotificationSent(channel models.NotificationChannel, eventType models.EventType, status models.NotificationStatus) {
	m.NotificationsSentTotal.WithLabelValues(
		string(channel),
		string(eventType),
		string(status),
	).Inc()
}

// RecordNotificationDeliveryDuration records notification delivery duration
func (m *Metrics) RecordNotificationDeliveryDuration(channel models.NotificationChannel, duration time.Duration) {
	m.NotificationsDeliveryDuration.WithLabelValues(string(channel)).Observe(duration.Seconds())
}

// RecordNotificationRetry records a notification retry
func (m *Metrics) RecordNotificationRetry(channel models.NotificationChannel, attempt int) {
	m.NotificationsRetryTotal.WithLabelValues(
		string(channel),
		fmt.Sprintf("%d", attempt),
	).Inc()
}

// RecordNotificationDLQ records a notification sent to DLQ
func (m *Metrics) RecordNotificationDLQ(eventType models.EventType) {
	m.NotificationsDLQTotal.WithLabelValues(string(eventType)).Inc()
}

// RecordNotificationBatched records a notification batched
func (m *Metrics) RecordNotificationBatched() {
	m.NotificationsBatchedTotal.Inc()
}

// RecordPreferencesUpdated records a preference update
func (m *Metrics) RecordPreferencesUpdated() {
	m.NotificationPreferencesUpdatedTotal.Inc()
}

// RecordChannelConnection records a channel connection
func (m *Metrics) RecordChannelConnection(channel models.NotificationChannel, count float64) {
	m.ChannelConnectionsTotal.WithLabelValues(string(channel)).Set(count)
}

// RecordChannelError records a channel error
func (m *Metrics) RecordChannelError(channel models.NotificationChannel, errorType string) {
	m.ChannelErrorsTotal.WithLabelValues(string(channel), errorType).Inc()
}

// RecordBatchSize records batch size
func (m *Metrics) RecordBatchSize(size int) {
	m.BatchSizeHistogram.Observe(float64(size))
}

// RecordBatchProcessingDuration records batch processing duration
func (m *Metrics) RecordBatchProcessingDuration(duration time.Duration) {
	m.BatchProcessingDuration.Observe(duration.Seconds())
}

// RecordDLQEntries records DLQ entries count
func (m *Metrics) RecordDLQEntries(count int64) {
	m.DLQEntriesTotal.Set(float64(count))
}

// RecordDLQRetryAttempt records a DLQ retry attempt
func (m *Metrics) RecordDLQRetryAttempt(eventType models.EventType, status string) {
	m.DLQRetryAttemptsTotal.WithLabelValues(string(eventType), status).Inc()
}

// RecordActiveConnections records active connections count
func (m *Metrics) RecordActiveConnections(count int) {
	m.ActiveConnections.Set(float64(count))
}

// RecordProcessingError records a processing error
func (m *Metrics) RecordProcessingError(service, errorType string) {
	m.ProcessingErrorsTotal.WithLabelValues(service, errorType).Inc()
}
