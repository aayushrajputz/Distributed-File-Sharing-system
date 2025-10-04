package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Cassandra write metrics
	CassandraWriteTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cassandra_write_total",
			Help: "Total number of Cassandra writes",
		},
		[]string{"operation", "status"},
	)

	// Cassandra query duration metrics
	CassandraQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cassandra_query_duration_seconds",
			Help:    "Cassandra query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	// Cassandra connection metrics
	CassandraConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cassandra_connections_active",
			Help: "Number of active Cassandra connections",
		},
	)

	// Cassandra error metrics
	CassandraErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cassandra_errors_total",
			Help: "Total number of Cassandra errors",
		},
		[]string{"operation", "error_type"},
	)

	// Cassandra event processing metrics
	CassandraEventsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cassandra_events_processed_total",
			Help: "Total number of events processed by Cassandra consumer",
		},
		[]string{"event_type", "status"},
	)

	// Cassandra consumer lag metrics
	CassandraConsumerLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cassandra_consumer_lag",
			Help: "Kafka consumer lag for Cassandra events",
		},
	)
)

// RecordCassandraWrite records a Cassandra write operation
func RecordCassandraWrite(operation, status string) {
	CassandraWriteTotal.WithLabelValues(operation, status).Inc()
}

// RecordCassandraQueryDuration records the duration of a Cassandra query
func RecordCassandraQueryDuration(operation string, duration float64) {
	CassandraQueryDuration.WithLabelValues(operation).Observe(duration)
}

// RecordCassandraError records a Cassandra error
func RecordCassandraError(operation, errorType string) {
	CassandraErrorsTotal.WithLabelValues(operation, errorType).Inc()
}

// RecordCassandraEventProcessed records a processed event
func RecordCassandraEventProcessed(eventType, status string) {
	CassandraEventsProcessed.WithLabelValues(eventType, status).Inc()
}

// SetCassandraConnectionsActive sets the number of active connections
func SetCassandraConnectionsActive(count float64) {
	CassandraConnectionsActive.Set(count)
}

// SetCassandraConsumerLag sets the consumer lag
func SetCassandraConsumerLag(lag float64) {
	CassandraConsumerLag.Set(lag)
}
