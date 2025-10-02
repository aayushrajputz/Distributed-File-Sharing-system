#!/bin/bash
# Script to set up NGINX Ingress Controller and deploy Ingress rules

set -e

NAMESPACE="${NAMESPACE:-file-sharing}"
INGRESS_NAMESPACE="ingress-nginx"

echo "==================================="
echo "Ingress Controller Setup Script"
echo "==================================="
echo "Target Namespace: $NAMESPACE"
echo "==================================="

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "Error: kubectl is not installed"
    exit 1
fi

# Check if helm is available
if ! command -v helm &> /dev/null; then
    echo "Warning: helm is not installed. Will use kubectl to check ingress controller."
    HELM_AVAILABLE=false
else
    HELM_AVAILABLE=true
fi

# Check if ingress controller is already installed
echo ""
echo "Checking for existing Ingress Controller..."
if kubectl get namespace $INGRESS_NAMESPACE &> /dev/null; then
    echo "✓ Ingress namespace exists"
    if kubectl get pods -n $INGRESS_NAMESPACE -l app.kubernetes.io/name=ingress-nginx &> /dev/null; then
        echo "✓ NGINX Ingress Controller appears to be installed"
        INGRESS_INSTALLED=true
    else
        INGRESS_INSTALLED=false
    fi
else
    INGRESS_INSTALLED=false
fi

# Install ingress controller if not present
if [ "$INGRESS_INSTALLED" = false ]; then
    echo ""
    echo "Installing NGINX Ingress Controller..."
    
    if [ "$HELM_AVAILABLE" = true ]; then
        echo "Using Helm to install..."
        helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
        helm repo update
        helm install ingress-nginx ingress-nginx/ingress-nginx \
            --namespace $INGRESS_NAMESPACE \
            --create-namespace \
            --set controller.service.type=LoadBalancer \
            --wait
        echo "✓ NGINX Ingress Controller installed via Helm"
    else
        echo "Using kubectl to install..."
        kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml
        echo "✓ NGINX Ingress Controller installed via kubectl"
        echo "  Waiting for controller to be ready..."
        kubectl wait --namespace $INGRESS_NAMESPACE \
            --for=condition=ready pod \
            --selector=app.kubernetes.io/component=controller \
            --timeout=120s
    fi
else
    echo "Skipping installation, ingress controller already present"
fi

# Verify ingress controller is running
echo ""
echo "Verifying Ingress Controller..."
kubectl get pods -n $INGRESS_NAMESPACE -l app.kubernetes.io/name=ingress-nginx

# Check if file-sharing namespace exists
echo ""
echo "Checking target namespace..."
if ! kubectl get namespace $NAMESPACE &> /dev/null; then
    echo "Error: Namespace '$NAMESPACE' does not exist"
    echo "Please create it first with: kubectl apply -f k8s/namespace.yaml"
    exit 1
fi
echo "✓ Namespace '$NAMESPACE' exists"

# Deploy API Gateway if not already deployed
echo ""
echo "Checking API Gateway deployment..."
if ! kubectl get deployment api-gateway -n $NAMESPACE &> /dev/null; then
    echo "API Gateway not found. Deploying..."
    kubectl apply -f k8s/api-gateway/
    echo "✓ API Gateway deployed"
else
    echo "✓ API Gateway already deployed"
fi

# Wait for API Gateway to be ready
echo "Waiting for API Gateway to be ready..."
kubectl wait --namespace $NAMESPACE \
    --for=condition=available \
    --timeout=120s \
    deployment/api-gateway

# Deploy Ingress rules
echo ""
echo "Deploying Ingress rules..."
kubectl apply -f k8s/ingress/ingress.yaml
echo "✓ Ingress rules deployed"

# Wait a moment for ingress to be configured
sleep 3

# Get Ingress information
echo ""
echo "==================================="
echo "Ingress Configuration Complete!"
echo "==================================="
echo ""
echo "Ingress Status:"
kubectl get ingress -n $NAMESPACE
echo ""

# Get Ingress IP/Hostname
echo "Getting Ingress address..."
INGRESS_ADDRESS=$(kubectl get ingress -n $NAMESPACE -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}')
if [ -z "$INGRESS_ADDRESS" ]; then
    INGRESS_ADDRESS=$(kubectl get ingress -n $NAMESPACE -o jsonpath='{.items[0].status.loadBalancer.ingress[0].hostname}')
fi

if [ -z "$INGRESS_ADDRESS" ]; then
    echo "⚠ Ingress address not yet assigned. This is normal for new installations."
    echo "  It may take a few minutes for the LoadBalancer to provision."
    echo ""
    echo "  Check status with:"
    echo "    kubectl get ingress -n $NAMESPACE"
    echo ""
    echo "  For local testing with port-forward:"
    echo "    kubectl port-forward svc/api-gateway 8080:8080 -n $NAMESPACE"
    echo "    Then access: http://localhost:8080/api/v1"
else
    echo ""
    echo "✓ Ingress Address: $INGRESS_ADDRESS"
    echo ""
    echo "  Access the API at:"
    echo "    http://$INGRESS_ADDRESS/api/v1"
    echo ""
    echo "  Test with:"
    echo "    curl http://$INGRESS_ADDRESS/api/v1/auth/health"
fi

echo ""
echo "==================================="
echo "Useful Commands:"
echo "==================================="
echo ""
echo "# View Ingress Controller logs:"
echo "kubectl logs -n $INGRESS_NAMESPACE -l app.kubernetes.io/name=ingress-nginx --tail=100 -f"
echo ""
echo "# View API Gateway logs:"
echo "kubectl logs -n $NAMESPACE -l app=api-gateway --tail=100 -f"
echo ""
echo "# Describe Ingress for details:"
echo "kubectl describe ingress -n $NAMESPACE"
echo ""
echo "# Port forward for local testing:"
echo "kubectl port-forward svc/api-gateway 8080:8080 -n $NAMESPACE"
echo ""
echo "==================================="

