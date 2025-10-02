#!/bin/bash
# Complete Kubernetes deployment script for distributed file-sharing platform

set -e

NAMESPACE="${NAMESPACE:-file-sharing}"
USE_STATEFULSET="${USE_STATEFULSET:-false}"

echo "==========================================="
echo "Kubernetes Deployment Script"
echo "==========================================="
echo "Namespace: $NAMESPACE"
echo "MongoDB: $([ "$USE_STATEFULSET" = "true" ] && echo "StatefulSet" || echo "Deployment")"
echo "==========================================="

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    exit 1
fi

# Check cluster connection
echo ""
echo "Checking cluster connection..."
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}Error: Cannot connect to Kubernetes cluster${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Connected to cluster${NC}"

# Function to wait for pods
wait_for_pods() {
    local label=$1
    local timeout=${2:-120}
    echo "  Waiting for pods with label $label..."
    kubectl wait --for=condition=ready pod -l "$label" -n "$NAMESPACE" --timeout="${timeout}s" || true
}

# Function to check deployment
check_deployment() {
    local name=$1
    echo "  Checking $name..."
    kubectl get pods -n "$NAMESPACE" -l "app=$name"
}

# Step 1: Create Namespace
echo ""
echo "Step 1: Creating namespace..."
kubectl apply -f k8s/namespace.yaml
echo -e "${GREEN}✓ Namespace created/verified${NC}"

# Step 2: Deploy Infrastructure - Secrets First
echo ""
echo "Step 2: Deploying Secrets..."
kubectl apply -f k8s/mongodb/mongodb-secret.yaml
kubectl apply -f k8s/minio/minio-secret.yaml
kubectl apply -f k8s/auth-service/auth-service-secret.yaml
kubectl apply -f k8s/file-service/file-service-secret.yaml
kubectl apply -f k8s/notification-service/notification-service-secret.yaml
echo -e "${GREEN}✓ Secrets deployed${NC}"

# Step 3: Deploy MongoDB
echo ""
echo "Step 3: Deploying MongoDB..."
if [ "$USE_STATEFULSET" = "true" ]; then
    echo "  Using StatefulSet..."
    kubectl apply -f k8s/mongodb/mongodb-statefulset.yaml
else
    echo "  Using Deployment..."
    kubectl apply -f k8s/mongodb/mongodb-deployment.yaml
fi
wait_for_pods "app=mongodb" 180
check_deployment "mongodb"
echo -e "${GREEN}✓ MongoDB deployed${NC}"

# Step 4: Deploy Zookeeper
echo ""
echo "Step 4: Deploying Zookeeper..."
kubectl apply -f k8s/zookeeper/zookeeper-deployment.yaml
wait_for_pods "app=zookeeper" 120
check_deployment "zookeeper"
echo -e "${GREEN}✓ Zookeeper deployed${NC}"

# Step 5: Deploy Kafka
echo ""
echo "Step 5: Deploying Kafka..."
kubectl apply -f k8s/kafka/kafka-deployment.yaml
echo "  Waiting for Kafka (this may take 2-3 minutes)..."
wait_for_pods "app=kafka" 240
check_deployment "kafka"
echo -e "${GREEN}✓ Kafka deployed${NC}"

# Step 6: Create Kafka Topics
echo ""
echo "Step 6: Creating Kafka topics..."
sleep 10  # Give Kafka a moment to fully initialize
KAFKA_POD=$(kubectl get pods -n "$NAMESPACE" -l app=kafka -o jsonpath='{.items[0].metadata.name}')
if [ -n "$KAFKA_POD" ]; then
    echo "  Creating file-events topic..."
    kubectl exec -n "$NAMESPACE" "$KAFKA_POD" -- kafka-topics --create \
        --topic file-events \
        --bootstrap-server localhost:9092 \
        --partitions 3 \
        --replication-factor 1 \
        --if-not-exists || echo "  Topic may already exist"
    
    echo "  Listing topics:"
    kubectl exec -n "$NAMESPACE" "$KAFKA_POD" -- \
        kafka-topics --list --bootstrap-server localhost:9092
    echo -e "${GREEN}✓ Kafka topics created${NC}"
else
    echo -e "${YELLOW}⚠ Could not find Kafka pod, skipping topic creation${NC}"
fi

# Step 7: Deploy MinIO
echo ""
echo "Step 7: Deploying MinIO..."
kubectl apply -f k8s/minio/minio-deployment.yaml
wait_for_pods "app=minio" 120
check_deployment "minio"
echo -e "${GREEN}✓ MinIO deployed${NC}"

# Step 8: Deploy ConfigMaps
echo ""
echo "Step 8: Deploying ConfigMaps..."
kubectl apply -f k8s/auth-service/auth-service-configmap.yaml
kubectl apply -f k8s/file-service/file-service-configmap.yaml
kubectl apply -f k8s/notification-service/notification-service-configmap.yaml
kubectl apply -f k8s/api-gateway/api-gateway-configmap.yaml
echo -e "${GREEN}✓ ConfigMaps deployed${NC}"

# Step 9: Deploy Microservices
echo ""
echo "Step 9: Deploying Microservices..."

echo "  9.1: Auth Service..."
kubectl apply -f k8s/auth-service/auth-service-deployment.yaml
wait_for_pods "app=auth-service" 120

echo "  9.2: File Service..."
kubectl apply -f k8s/file-service/file-service-deployment.yaml
wait_for_pods "app=file-service" 120

echo "  9.3: Notification Service..."
kubectl apply -f k8s/notification-service/notification-service-deployment.yaml
wait_for_pods "app=notification-service" 120

echo "  9.4: API Gateway..."
kubectl apply -f k8s/api-gateway/api-gateway-deployment.yaml
wait_for_pods "app=api-gateway" 120

echo -e "${GREEN}✓ All microservices deployed${NC}"

# Step 10: Deploy Ingress (optional)
echo ""
read -p "Do you want to deploy Ingress? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Step 10: Deploying Ingress..."
    
    # Check if ingress controller exists
    if ! kubectl get namespace ingress-nginx &> /dev/null; then
        echo "  Ingress controller not found. Please run ./scripts/setup-ingress.sh first"
    else
        kubectl apply -f k8s/ingress/ingress.yaml
        echo -e "${GREEN}✓ Ingress deployed${NC}"
    fi
else
    echo "  Skipping Ingress deployment"
fi

# Step 11: Verification
echo ""
echo "==========================================="
echo "Deployment Complete!"
echo "==========================================="
echo ""
echo "Verifying deployment..."
echo ""

echo "Pods:"
kubectl get pods -n "$NAMESPACE"
echo ""

echo "Services:"
kubectl get svc -n "$NAMESPACE"
echo ""

echo "PVCs:"
kubectl get pvc -n "$NAMESPACE"
echo ""

echo "Ingress:"
kubectl get ingress -n "$NAMESPACE" 2>/dev/null || echo "  No Ingress configured"
echo ""

# Check for any issues
echo "Checking for issues..."
PROBLEM_PODS=$(kubectl get pods -n "$NAMESPACE" --field-selector=status.phase!=Running -o name 2>/dev/null | wc -l)
if [ "$PROBLEM_PODS" -gt 0 ]; then
    echo -e "${YELLOW}⚠ Warning: $PROBLEM_PODS pod(s) not in Running state${NC}"
    kubectl get pods -n "$NAMESPACE" --field-selector=status.phase!=Running
    echo ""
    echo "Check logs with:"
    echo "  kubectl logs -n $NAMESPACE <pod-name>"
else
    echo -e "${GREEN}✓ All pods are running${NC}"
fi

echo ""
echo "==========================================="
echo "Next Steps:"
echo "==========================================="
echo ""
echo "1. Test API Gateway:"
echo "   kubectl port-forward svc/api-gateway 8080:8080 -n $NAMESPACE"
echo "   curl http://localhost:8080/health"
echo ""
echo "2. View logs:"
echo "   kubectl logs -n $NAMESPACE -l app=api-gateway --tail=50 -f"
echo ""
echo "3. Check all resources:"
echo "   kubectl get all -n $NAMESPACE"
echo ""
echo "4. Access services (if Ingress is configured):"
INGRESS_IP=$(kubectl get ingress -n "$NAMESPACE" -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}' 2>/dev/null)
if [ -n "$INGRESS_IP" ]; then
    echo "   curl http://$INGRESS_IP/api/v1/auth/health"
else
    echo "   Ingress IP not yet assigned (check: kubectl get ingress -n $NAMESPACE)"
fi
echo ""
echo "5. For detailed deployment guide, see:"
echo "   docs/KUBERNETES_DEPLOYMENT.md"
echo ""
echo "==========================================="

