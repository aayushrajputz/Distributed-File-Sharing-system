package cassandra

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Hosts       []string      // Comma-separated in env: CASSANDRA_HOSTS
	Port        int           // CASSANDRA_PORT (default: 9042)
	Keyspace    string        // CASSANDRA_KEYSPACE
	Username    string        // CASSANDRA_USER (optional)
	Password    string        // CASSANDRA_PASSWORD (optional)
	Consistency string        // CASSANDRA_CONSISTENCY (default: LOCAL_QUORUM)
	Timeout     time.Duration // CASSANDRA_TIMEOUT (default: 10s)
	NumConns    int           // CASSANDRA_NUM_CONNS (default: 2)
	EnableTLS   bool          // CASSANDRA_TLS_ENABLED (default: false)
}

// Client interface defines the methods for Cassandra operations
type Client interface {
	GetSession() *gocql.Session
	Close()
	HealthCheck(ctx context.Context) error
}

type ClientImpl struct {
	session *gocql.Session
	config  *Config
	logger  *logrus.Logger
}

// NewClient creates a new Cassandra client with connection pooling and retries
func NewClient(cfg *Config, logger *logrus.Logger) (Client, error) {
	// First, create a cluster without keyspace to check connectivity
	cluster := gocql.NewCluster(cfg.Hosts...)
	cluster.Port = cfg.Port
	cluster.Timeout = cfg.Timeout
	cluster.ConnectTimeout = 10 * time.Second
	cluster.NumConns = cfg.NumConns

	// Set consistency level
	consistency := gocql.LocalQuorum
	switch strings.ToUpper(cfg.Consistency) {
	case "ONE":
		consistency = gocql.One
	case "QUORUM":
		consistency = gocql.Quorum
	case "LOCAL_QUORUM":
		consistency = gocql.LocalQuorum
	case "ALL":
		consistency = gocql.All
	}
	cluster.Consistency = consistency

	// Authentication
	if cfg.Username != "" && cfg.Password != "" {
		cluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	// TLS configuration
	if cfg.EnableTLS {
		cluster.SslOpts = &gocql.SslOptions{
			EnableHostVerification: true,
		}
	}

	// Retry policy with exponential backoff
	cluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
		NumRetries: 3,
		Min:        100 * time.Millisecond,
		Max:        10 * time.Second,
	}

	// Connection pooling
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())

	// Create session without keyspace first
	session, err := cluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create Cassandra session: %w", err)
	}

	// Create keyspace if it doesn't exist
	if err := createKeyspaceIfNotExists(session, cfg.Keyspace, logger); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create keyspace: %w", err)
	}

	// Close the initial session and create a new cluster with the keyspace
	session.Close()

	// Create a new cluster configuration for the keyspace session
	keyspaceCluster := gocql.NewCluster(cfg.Hosts...)
	keyspaceCluster.Port = cfg.Port
	keyspaceCluster.Keyspace = cfg.Keyspace
	keyspaceCluster.Timeout = cfg.Timeout
	keyspaceCluster.ConnectTimeout = 10 * time.Second
	keyspaceCluster.NumConns = cfg.NumConns
	keyspaceCluster.Consistency = consistency

	// Authentication
	if cfg.Username != "" && cfg.Password != "" {
		keyspaceCluster.Authenticator = gocql.PasswordAuthenticator{
			Username: cfg.Username,
			Password: cfg.Password,
		}
	}

	// TLS configuration
	if cfg.EnableTLS {
		keyspaceCluster.SslOpts = &gocql.SslOptions{
			EnableHostVerification: true,
		}
	}

	// Retry policy with exponential backoff
	keyspaceCluster.RetryPolicy = &gocql.ExponentialBackoffRetryPolicy{
		NumRetries: 3,
		Min:        100 * time.Millisecond,
		Max:        10 * time.Second,
	}

	// Connection pooling - use RoundRobinHostPolicy instead of TokenAwareHostPolicy
	keyspaceCluster.PoolConfig.HostSelectionPolicy = gocql.RoundRobinHostPolicy()

	session, err = keyspaceCluster.CreateSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create Cassandra session with keyspace: %w", err)
	}

	// Create tables if they don't exist
	if err := createTablesIfNotExists(session, logger); err != nil {
		session.Close()
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	logger.Info("Cassandra client initialized successfully")

	return &ClientImpl{
		session: session,
		config:  cfg,
		logger:  logger,
	}, nil
}

// createKeyspaceIfNotExists creates the keyspace if it doesn't exist
func createKeyspaceIfNotExists(session *gocql.Session, keyspace string, logger *logrus.Logger) error {
	query := fmt.Sprintf(`
		CREATE KEYSPACE IF NOT EXISTS %s
		WITH replication = {
			'class': 'SimpleStrategy',
			'replication_factor': 1
		}
	`, keyspace)

	if err := session.Query(query).Exec(); err != nil {
		return fmt.Errorf("failed to create keyspace %s: %w", keyspace, err)
	}

	logger.WithField("keyspace", keyspace).Info("Keyspace created or already exists")
	return nil
}

// createTablesIfNotExists creates the required tables if they don't exist
func createTablesIfNotExists(session *gocql.Session, logger *logrus.Logger) error {
	// Create file_events table
	fileEventsQuery := `
		CREATE TABLE IF NOT EXISTS file_events (
			user_id text,
			event_ts timestamp,
			event_id uuid,
			file_id uuid,
			action text,
			status text,
			file_name text,
			file_size bigint,
			metadata text,
			PRIMARY KEY (user_id, event_ts, event_id)
		) WITH CLUSTERING ORDER BY (event_ts DESC, event_id ASC)
	`

	if err := session.Query(fileEventsQuery).Exec(); err != nil {
		return fmt.Errorf("failed to create file_events table: %w", err)
	}

	// Create file_versions table
	fileVersionsQuery := `
		CREATE TABLE IF NOT EXISTS file_versions (
			file_id uuid,
			version int,
			file_name text,
			file_size bigint,
			content_type text,
			storage_path text,
			checksum text,
			uploaded_at timestamp,
			uploaded_by text,
			metadata text,
			PRIMARY KEY (file_id, version)
		) WITH CLUSTERING ORDER BY (version DESC)
	`

	if err := session.Query(fileVersionsQuery).Exec(); err != nil {
		return fmt.Errorf("failed to create file_versions table: %w", err)
	}

	// Create files_metadata table (optional)
	filesMetadataQuery := `
		CREATE TABLE IF NOT EXISTS files_metadata (
			file_id uuid PRIMARY KEY,
			file_name text,
			owner_id text,
			file_size bigint,
			content_type text,
			status text,
			created_at timestamp,
			updated_at timestamp,
			deleted_at timestamp,
			storage_path text
		)
	`

	if err := session.Query(filesMetadataQuery).Exec(); err != nil {
		return fmt.Errorf("failed to create files_metadata table: %w", err)
	}

	logger.Info("All Cassandra tables created or already exist")
	return nil
}

// GetSession returns the underlying gocql session
func (c *ClientImpl) GetSession() *gocql.Session {
	return c.session
}

// Close gracefully closes the Cassandra session
func (c *ClientImpl) Close() {
	if c.session != nil {
		c.session.Close()
		c.logger.Info("Cassandra session closed")
	}
}

// HealthCheck verifies Cassandra connectivity
func (c *ClientImpl) HealthCheck(ctx context.Context) error {
	query := c.session.Query("SELECT now() FROM system.local")
	if err := query.WithContext(ctx).Exec(); err != nil {
		return fmt.Errorf("cassandra health check failed: %w", err)
	}
	return nil
}
