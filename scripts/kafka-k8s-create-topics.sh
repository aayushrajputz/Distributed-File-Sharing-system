#!/bin/bash
# Script to create Kafka topics in Kubernetes

set -e

NAMESPACE="${NAMESPACE:-file-sharing}"
PARTITIONS="${PARTITIONS:-3}"
REPLICATION_FACTOR="${REPLICATION_FACTOR:-1}"

echo "==================================="
echo "Creating Kafka Topics in Kubernetes"
echo "==================================="
echo "Namespace: $NAMESPACE"
echo "Partitions: $PARTITIONS"
echo "Replication Factor: $REPLICATION_FACTOR"
echo "==================================="

# Check if namespace exists
if ! kubectl get namespace "$NAMESPACE" > /dev/null 2>&1; then
    echo "Error: Namespace '$NAMESPACE' does not exist"
    echo "Please create it first with: kubectl apply -f k8s/namespace.yaml"
    exit 1
fi
echo "✓ Namespace exists"

# Check if Kafka pod is running
echo "Checking Kafka pod..."
if ! kubectl get pods -n "$NAMESPACE" -l app=kafka | grep -q "Running"; then
    echo "Error: Kafka pod is not running in namespace '$NAMESPACE'"
    echo "Please deploy Kafka first with: kubectl apply -f k8s/kafka/"
    exit 1
fi
echo "✓ Kafka pod is running"

# Get Kafka pod name
KAFKA_POD=$(kubectl get pods -n "$NAMESPACE" -l app=kafka -o jsonpath='{.items[0].metadata.name}')
echo "Using Kafka pod: $KAFKA_POD"

# Create topics
echo ""
echo "Creating file-events topic..."
kubectl exec -n "$NAMESPACE" "$KAFKA_POD" -- kafka-topics --create \
    --topic file-events \
    --bootstrap-server localhost:9092 \
    --partitions "$PARTITIONS" \
    --replication-factor "$REPLICATION_FACTOR" \
    --if-not-exists

echo ""
echo "✓ Topics created successfully"
echo ""
echo "All topics:"
kubectl exec -n "$NAMESPACE" "$KAFKA_POD" -- \
    kafka-topics --list --bootstrap-server localhost:9092

echo ""
echo "Topic details:"
kubectl exec -n "$NAMESPACE" "$KAFKA_POD" -- \
    kafka-topics --describe \
    --topic file-events \
    --bootstrap-server localhost:9092

echo ""
echo "==================================="
echo "Done!"
echo "==================================="

