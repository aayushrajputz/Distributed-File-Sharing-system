#!/bin/bash
# Script to create required Kafka topics for the distributed file-sharing platform

set -e

# Configuration
KAFKA_HOST="${KAFKA_HOST:-localhost:9092}"
PARTITIONS="${PARTITIONS:-3}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-1}"

echo "==================================="
echo "Kafka Topic Creation Script"
echo "==================================="
echo "Kafka Host: $KAFKA_HOST"
echo "Partitions: $PARTITIONS"
echo "Replication Factor: $REPLICATION_FACTOR"
echo "==================================="

# Check if Kafka is reachable
echo "Checking Kafka connection..."
if ! kafka-broker-api-versions --bootstrap-server "$KAFKA_HOST" > /dev/null 2>&1; then
    echo "Error: Cannot connect to Kafka at $KAFKA_HOST"
    echo "Please ensure Kafka is running and accessible."
    exit 1
fi
echo "✓ Kafka is reachable"

# Function to create topic
create_topic() {
    local topic_name=$1
    
    echo ""
    echo "Checking topic: $topic_name"
    
    # Check if topic already exists
    if kafka-topics --list --bootstrap-server "$KAFKA_HOST" | grep -q "^${topic_name}$"; then
        echo "  ⚠ Topic '$topic_name' already exists. Skipping..."
        
        # Display topic details
        echo "  Topic details:"
        kafka-topics --describe --topic "$topic_name" --bootstrap-server "$KAFKA_HOST" | sed 's/^/    /'
    else
        echo "  Creating topic '$topic_name'..."
        kafka-topics --create \
            --topic "$topic_name" \
            --bootstrap-server "$KAFKA_HOST" \
            --partitions "$PARTITIONS" \
            --replication-factor "$REPLICATION_FACTOR"
        
        echo "  ✓ Topic '$topic_name' created successfully"
        
        # Display topic details
        echo "  Topic details:"
        kafka-topics --describe --topic "$topic_name" --bootstrap-server "$KAFKA_HOST" | sed 's/^/    /'
    fi
}

# Create required topics
echo ""
echo "Creating topics..."
echo "-----------------------------------"

# Main topic for file events
create_topic "file-events"

# Optional: Create additional topics if needed
# create_topic "user-activity"
# create_topic "file-downloads"
# create_topic "file-shares"

echo ""
echo "==================================="
echo "Topic Creation Complete!"
echo "==================================="
echo ""
echo "All topics:"
kafka-topics --list --bootstrap-server "$KAFKA_HOST"
echo ""
echo "==================================="

