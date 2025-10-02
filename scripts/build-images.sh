#!/bin/bash

# Build and push Docker images for all services

set -e

# Configuration
REGISTRY=${DOCKER_REGISTRY:-"your-registry"}
TAG=${IMAGE_TAG:-"latest"}

echo "Building Docker images with tag: $TAG"

# Build Auth Service
echo "Building Auth Service..."
docker build -t $REGISTRY/auth-service:$TAG ./services/auth-service
docker push $REGISTRY/auth-service:$TAG

# Build File Service
echo "Building File Service..."
docker build -t $REGISTRY/file-service:$TAG ./services/file-service
docker push $REGISTRY/file-service:$TAG

# Build Notification Service
echo "Building Notification Service..."
docker build -t $REGISTRY/notification-service:$TAG ./services/notification-service
docker push $REGISTRY/notification-service:$TAG

# Build Frontend
echo "Building Frontend..."
docker build -t $REGISTRY/frontend:$TAG ./frontend
docker push $REGISTRY/frontend:$TAG

echo "All images built and pushed successfully!"

