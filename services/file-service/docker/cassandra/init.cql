-- Cassandra Schema for Distributed File Sharing Platform
-- This script initializes the keyspace and tables for file events and version history

-- Keyspace creation
-- DEV: SimpleStrategy with RF=1 (single node)
-- PROD: Use NetworkTopologyStrategy with RF=3 per DC
CREATE KEYSPACE IF NOT EXISTS file_service
WITH replication = {
  'class': 'SimpleStrategy',
  'replication_factor': 1
};

-- For production multi-DC setup, use:
-- CREATE KEYSPACE IF NOT EXISTS file_service
-- WITH replication = {
--   'class': 'NetworkTopologyStrategy',
--   'dc1': 3,
--   'dc2': 3
-- };

USE file_service;

-- Table 1: File Events (append-only audit log)
-- Query pattern: Get all events for a user, ordered by time (newest first)
-- Partition key: user_id (distributes load by user)
-- Clustering keys: event_ts DESC, event_id (for uniqueness and ordering)
CREATE TABLE IF NOT EXISTS file_events (
  user_id text,
  event_ts timestamp,
  event_id uuid,
  file_id uuid,
  action text,           -- 'upload', 'delete', 'restore', 'permanent_delete', 'download'
  status text,           -- 'success', 'failed', 'pending'
  file_name text,
  file_size bigint,
  metadata text,         -- JSON string for extensibility
  PRIMARY KEY (user_id, event_ts, event_id)
) WITH CLUSTERING ORDER BY (event_ts DESC, event_id ASC)
  AND compaction = {'class': 'TimeWindowCompactionStrategy', 'compaction_window_size': 1, 'compaction_window_unit': 'DAYS'}
  AND gc_grace_seconds = 864000
  AND comment = 'Append-only event log for file operations. Query by user_id and time range.';

-- Optional: Add TTL for automatic cleanup after 90 days
-- ALTER TABLE file_events WITH default_time_to_live = 7776000;

-- Table 2: File Versions (version history per file)
-- Query pattern: Get all versions for a file, ordered by version (newest first)
-- Partition key: file_id
-- Clustering key: version DESC
CREATE TABLE IF NOT EXISTS file_versions (
  file_id uuid,
  version int,
  file_name text,
  file_size bigint,
  content_type text,
  storage_path text,
  checksum text,
  uploaded_at timestamp,
  uploaded_by text,      -- user_id
  metadata text,         -- JSON string
  PRIMARY KEY (file_id, version)
) WITH CLUSTERING ORDER BY (version DESC)
  AND compaction = {'class': 'SizeTieredCompactionStrategy'}
  AND comment = 'Version history for files. Each upload creates a new version.';

-- Table 3: File Metadata Mirror (optional - for fast Cassandra-only queries)
-- Only use if you need to query file metadata without hitting MongoDB
-- Otherwise, keep MongoDB as single source of truth
CREATE TABLE IF NOT EXISTS files_metadata (
  file_id uuid PRIMARY KEY,
  file_name text,
  owner_id text,
  file_size bigint,
  content_type text,
  status text,           -- 'available', 'deleted', 'processing'
  created_at timestamp,
  updated_at timestamp,
  deleted_at timestamp,
  storage_path text
) WITH comment = 'Optional mirror of active file metadata from MongoDB. Use only if needed for Cassandra-native queries.';

-- Secondary index for querying files by owner (use sparingly in production)
-- CREATE INDEX IF NOT EXISTS files_by_owner ON files_metadata (owner_id);

-- Production tuning recommendations (document in README):
-- 1. Compaction: TimeWindowCompactionStrategy for time-series data (file_events)
-- 2. Consistency: Use LOCAL_QUORUM for reads/writes in multi-DC
-- 3. TTL: Consider adding TTL to file_events for automatic cleanup
-- 4. Monitoring: Track tombstone count, compaction stats, read/write latencies
-- 5. Repair: Run nodetool repair weekly on production clusters
