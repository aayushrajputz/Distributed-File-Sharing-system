#!/bin/bash
# Script to create Kafka topics inside Docker container

set -e

CONTAINER_NAME="${CONTAINER_NAME:-kafka}"
PARTITIONS="${PARTITIONS:-3}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-1}"

echo "==================================="
echo "Creating Kafka Topics in Docker"
echo "==================================="
echo "Container: $CONTAINER_NAME"
echo "Partitions: $PARTITIONS"
echo "Replication Factor: $REPLICATION_FACTOR"
echo "==================================="

# Check if container is running
if ! docker ps | grep -q "$CONTAINER_NAME"; then
    echo "Error: Container '$CONTAINER_NAME' is not running"
    echo "Please start the container first with: docker-compose up -d kafka"
    exit 1
fi
echo "✓ Container is running"

# Create topics
echo ""
echo "Creating file-events topic..."
docker exec "$CONTAINER_NAME" kafka-topics --create \
    --topic file-events \
    --bootstrap-server localhost:9092 \
    --partitions "$PARTITIONS" \
    --replication-factor "$REPLICATION_FACTOR" \
    --if-not-exists

echo ""
echo "✓ Topics created successfully"
echo ""
echo "All topics:"
docker exec "$CONTAINER_NAME" kafka-topics --list --bootstrap-server localhost:9092

echo ""
echo "Topic details:"
docker exec "$CONTAINER_NAME" kafka-topics --describe \
    --topic file-events \
    --bootstrap-server localhost:9092

echo ""
echo "==================================="
echo "Done!"
echo "==================================="

