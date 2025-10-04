#!/bin/bash

# Build and Run Script for Distributed File Sharing Platform
# This script builds all services and runs them with proper orchestration

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}=== $1 ===${NC}"
}

print_info() {
    echo -e "${CYAN}$1${NC}"
}

print_warning() {
    echo -e "${YELLOW}$1${NC}"
}

print_error() {
    echo -e "${RED}$1${NC}"
}

# Function to check if Docker is running
check_docker() {
    if ! docker version >/dev/null 2>&1; then
        print_error "Docker is not running. Please start Docker Desktop."
        exit 1
    fi
    print_info "Docker is running ✓"
}

# Function to clean up containers and volumes
clean_environment() {
    print_info "Cleaning up environment..."
    docker-compose down -v --remove-orphans
    docker system prune -f
    print_info "Environment cleaned ✓"
}

# Function to build all services
build_services() {
    print_status "Building all services"
    
    # Generate protobuf files first
    print_info "Generating protobuf files..."
    cd scripts
    chmod +x generate-proto.sh
    ./generate-proto.sh
    cd ..
    
    # Build Docker images
    print_info "Building Docker images..."
    docker-compose build --no-cache --parallel
    
    if [ $? -ne 0 ]; then
        print_error "Build failed!"
        exit 1
    fi
    
    print_info "All services built successfully ✓"
}

# Function to run services
start_services() {
    print_status "Starting services"
    
    # Start infrastructure services first
    print_info "Starting infrastructure services (MongoDB, Redis, MinIO, Kafka, Cassandra)..."
    docker-compose up -d mongodb redis minio zookeeper kafka cassandra
    
    # Wait for infrastructure to be ready
    print_info "Waiting for infrastructure services to be ready..."
    sleep 30
    
    # Check Cassandra health
    print_info "Checking Cassandra health..."
    max_retries=10
    retry_count=0
    while [ $retry_count -lt $max_retries ]; do
        if docker exec cassandra cqlsh -e "describe cluster" >/dev/null 2>&1; then
            print_info "Cassandra is ready ✓"
            break
        else
            retry_count=$((retry_count + 1))
            print_warning "Cassandra not ready yet, waiting... (attempt $retry_count/$max_retries)"
            sleep 10
        fi
    done
    
    if [ $retry_count -eq $max_retries ]; then
        print_error "Cassandra failed to start properly!"
        exit 1
    fi
    
    # Start Cassandra initialization
    print_info "Running Cassandra initialization..."
    docker-compose up cassandra-init
    
    # Start application services
    print_info "Starting application services..."
    docker-compose up -d auth-service file-service notification-service billing-service api-gateway share-tracker frontend
    
    # Wait for services to be ready
    print_info "Waiting for application services to be ready..."
    sleep 20
    
    # Check service health
    print_info "Checking service health..."
    
    # Check MongoDB
    if docker exec mongodb mongosh --eval "db.runCommand({ping: 1})" >/dev/null 2>&1; then
        print_info "✓ MongoDB is healthy"
    else
        print_warning "✗ MongoDB is not responding"
    fi
    
    # Check Redis
    if docker exec redis redis-cli ping >/dev/null 2>&1; then
        print_info "✓ Redis is healthy"
    else
        print_warning "✗ Redis is not responding"
    fi
    
    # Check MinIO
    if curl -f http://localhost:9000/minio/health/live >/dev/null 2>&1; then
        print_info "✓ MinIO is healthy"
    else
        print_warning "✗ MinIO is not responding"
    fi
    
    # Check Kafka
    if docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1; then
        print_info "✓ Kafka is healthy"
    else
        print_warning "✗ Kafka is not responding"
    fi
    
    # Check Cassandra
    if docker exec cassandra cqlsh -e "describe cluster" >/dev/null 2>&1; then
        print_info "✓ Cassandra is healthy"
    else
        print_warning "✗ Cassandra is not responding"
    fi
    
    # Check Auth Service
    if curl -f http://localhost:8081/health >/dev/null 2>&1; then
        print_info "✓ Auth Service is healthy"
    else
        print_warning "✗ Auth Service is not responding"
    fi
    
    # Check File Service
    if curl -f http://localhost:8082/health >/dev/null 2>&1; then
        print_info "✓ File Service is healthy"
    else
        print_warning "✗ File Service is not responding"
    fi
    
    # Check Notification Service
    if curl -f http://localhost:8084/api/v1/health >/dev/null 2>&1; then
        print_info "✓ Notification Service is healthy"
    else
        print_warning "✗ Notification Service is not responding"
    fi
    
    # Check Billing Service
    if curl -f http://localhost:8086/health >/dev/null 2>&1; then
        print_info "✓ Billing Service is healthy"
    else
        print_warning "✗ Billing Service is not responding"
    fi
    
    # Check API Gateway
    if curl -f http://localhost:8080/health >/dev/null 2>&1; then
        print_info "✓ API Gateway is healthy"
    else
        print_warning "✗ API Gateway is not responding"
    fi
    
    # Check Frontend
    if curl -f http://localhost:3000 >/dev/null 2>&1; then
        print_info "✓ Frontend is healthy"
    else
        print_warning "✗ Frontend is not responding"
    fi
    
    echo ""
    print_status "Service URLs"
    print_info "Frontend: http://localhost:3000"
    print_info "API Gateway: http://localhost:8080"
    print_info "Auth Service: http://localhost:8081"
    print_info "File Service: http://localhost:8082"
    print_info "Notification Service: http://localhost:8084"
    print_info "Billing Service: http://localhost:8086"
    print_info "MinIO Console: http://localhost:9001 (minioadmin/minioadmin)"
    
    print_info "All services are running ✓"
}

# Function to show logs
show_logs() {
    print_info "Showing service logs..."
    docker-compose logs -f
}

# Function to stop services
stop_services() {
    print_info "Stopping all services..."
    docker-compose down
    print_info "All services stopped ✓"
}

# Main execution
print_status "Distributed File Sharing Platform - Build & Run Script"

# Check Docker
check_docker

# Handle command line arguments
case "${1:-}" in
    "clean")
        clean_environment
        exit 0
        ;;
    "stop")
        stop_services
        exit 0
        ;;
    "logs")
        show_logs
        exit 0
        ;;
    "build")
        build_services
        exit 0
        ;;
    "run")
        start_services
        exit 0
        ;;
    *)
        # Default: Build and Run
        print_info "Building and running all services..."
        build_services
        start_services
        
        echo ""
        print_status "Usage"
        print_info "To view logs: ./build-and-run.sh logs"
        print_info "To stop services: ./build-and-run.sh stop"
        print_info "To clean environment: ./build-and-run.sh clean"
        print_info "To build only: ./build-and-run.sh build"
        print_info "To run only: ./build-and-run.sh run"
        ;;
esac
