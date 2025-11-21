# Build and Run Script for Distributed File Sharing Platform
# This script builds all services and runs them with proper orchestration

param(
    [switch]$BuildOnly,
    [switch]$RunOnly,
    [switch]$Clean,
    [switch]$Logs,
    [switch]$Stop
)

Write-Host "=== Distributed File Sharing Platform - Build & Run Script ===" -ForegroundColor Green

# Function to check if Docker is running
function Test-Docker {
    try {
        docker version | Out-Null
        return $true
    } catch {
        Write-Host "Docker is not running. Please start Docker Desktop." -ForegroundColor Red
        return $false
    }
}

# Function to clean up containers and volumes
function Clean-Environment {
    Write-Host "Cleaning up environment..." -ForegroundColor Yellow
    docker-compose down -v --remove-orphans
    docker system prune -f
    Write-Host "Environment cleaned." -ForegroundColor Green
}

# Function to build all services
function Build-Services {
    Write-Host "Building all services..." -ForegroundColor Yellow
    
    # Generate protobuf files first
    Write-Host "Generating protobuf files..." -ForegroundColor Cyan
    Set-Location scripts
    .\generate-all-proto.ps1
    Set-Location ..
    
    # Build Docker images
    Write-Host "Building Docker images..." -ForegroundColor Cyan
    docker-compose build --no-cache --parallel
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Build failed!" -ForegroundColor Red
        exit 1
    }
    
    Write-Host "All services built successfully!" -ForegroundColor Green
}

# Function to run services
function Start-Services {
    Write-Host "Starting services..." -ForegroundColor Yellow
    
    # Start infrastructure services first
    Write-Host "Starting infrastructure services (MongoDB, Redis, MinIO, Kafka, Cassandra)..." -ForegroundColor Cyan
    docker-compose up -d mongodb redis minio zookeeper kafka cassandra
    
    # Wait for infrastructure to be ready
    Write-Host "Waiting for infrastructure services to be ready..." -ForegroundColor Cyan
    Start-Sleep -Seconds 30
    
    # Check Cassandra health
    Write-Host "Checking Cassandra health..." -ForegroundColor Cyan
    $maxRetries = 10
    $retryCount = 0
    do {
        try {
            docker exec cassandra cqlsh -e "describe cluster" | Out-Null
            Write-Host "Cassandra is ready!" -ForegroundColor Green
            break
        } catch {
            $retryCount++
            Write-Host "Cassandra not ready yet, waiting... (attempt $retryCount/$maxRetries)" -ForegroundColor Yellow
            Start-Sleep -Seconds 10
        }
    } while ($retryCount -lt $maxRetries)
    
    if ($retryCount -eq $maxRetries) {
        Write-Host "Cassandra failed to start properly!" -ForegroundColor Red
        exit 1
    }
    
    # Start Cassandra initialization
    Write-Host "Running Cassandra initialization..." -ForegroundColor Cyan
    docker-compose up cassandra-init
    
    # Start application services
    Write-Host "Starting application services..." -ForegroundColor Cyan
    docker-compose up -d auth-service file-service notification-service billing-service api-gateway share-tracker frontend
    
    # Wait for services to be ready
    Write-Host "Waiting for application services to be ready..." -ForegroundColor Cyan
    Start-Sleep -Seconds 20
    
    # Check service health
    Write-Host "Checking service health..." -ForegroundColor Cyan
    
    # Check MongoDB
    try {
        docker exec mongodb mongosh --eval "db.runCommand({ping: 1})" | Out-Null
        Write-Host "✓ MongoDB is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ MongoDB is not responding" -ForegroundColor Red
    }
    
    # Check Redis
    try {
        docker exec redis redis-cli ping | Out-Null
        Write-Host "✓ Redis is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ Redis is not responding" -ForegroundColor Red
    }
    
    # Check MinIO
    try {
        Invoke-RestMethod -Uri "http://localhost:9000/minio/health/live" -Method GET | Out-Null
        Write-Host "✓ MinIO is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ MinIO is not responding" -ForegroundColor Red
    }
    
    # Check Kafka
    try {
        docker exec kafka kafka-topics --bootstrap-server localhost:9092 --list | Out-Null
        Write-Host "✓ Kafka is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ Kafka is not responding" -ForegroundColor Red
    }
    
    # Check Cassandra
    try {
        docker exec cassandra cqlsh -e "describe cluster" | Out-Null
        Write-Host "✓ Cassandra is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ Cassandra is not responding" -ForegroundColor Red
    }
    
    # Check Auth Service
    try {
        Invoke-RestMethod -Uri "http://localhost:8081/health" -Method GET | Out-Null
        Write-Host "✓ Auth Service is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ Auth Service is not responding" -ForegroundColor Red
    }
    
    # Check File Service
    try {
        Invoke-RestMethod -Uri "http://localhost:8082/health" -Method GET | Out-Null
        Write-Host "✓ File Service is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ File Service is not responding" -ForegroundColor Red
    }
    
    # Check Notification Service
    try {
        Invoke-RestMethod -Uri "http://localhost:8084/api/v1/health" -Method GET | Out-Null
        Write-Host "✓ Notification Service is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ Notification Service is not responding" -ForegroundColor Red
    }
    
    # Check Billing Service
    try {
        Invoke-RestMethod -Uri "http://localhost:8086/health" -Method GET | Out-Null
        Write-Host "✓ Billing Service is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ Billing Service is not responding" -ForegroundColor Red
    }
    
    # Check API Gateway
    try {
        Invoke-RestMethod -Uri "http://localhost:8080/health" -Method GET | Out-Null
        Write-Host "✓ API Gateway is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ API Gateway is not responding" -ForegroundColor Red
    }
    
    # Check Frontend
    try {
        Invoke-RestMethod -Uri "http://localhost:3000" -Method GET | Out-Null
        Write-Host "✓ Frontend is healthy" -ForegroundColor Green
    } catch {
        Write-Host "✗ Frontend is not responding" -ForegroundColor Red
    }
    
    Write-Host "`n=== Service URLs ===" -ForegroundColor Green
    Write-Host "Frontend: http://localhost:3000" -ForegroundColor Cyan
    Write-Host "API Gateway: http://localhost:8080" -ForegroundColor Cyan
    Write-Host "Auth Service: http://localhost:8081" -ForegroundColor Cyan
    Write-Host "File Service: http://localhost:8082" -ForegroundColor Cyan
    Write-Host "Notification Service: http://localhost:8084" -ForegroundColor Cyan
    Write-Host "Billing Service: http://localhost:8086" -ForegroundColor Cyan
    Write-Host "MinIO Console: http://localhost:9001 (minioadmin/minioadmin)" -ForegroundColor Cyan
    
    Write-Host "`nAll services are running!" -ForegroundColor Green
}

# Function to show logs
function Show-Logs {
    Write-Host "Showing service logs..." -ForegroundColor Yellow
    docker-compose logs -f
}

# Function to stop services
function Stop-Services {
    Write-Host "Stopping all services..." -ForegroundColor Yellow
    docker-compose down
    Write-Host "All services stopped." -ForegroundColor Green
}

# Main execution
if (-not (Test-Docker)) {
    exit 1
}

if ($Clean) {
    Clean-Environment
    exit 0
}

if ($Stop) {
    Stop-Services
    exit 0
}

if ($Logs) {
    Show-Logs
    exit 0
}

if ($BuildOnly) {
    Build-Services
    exit 0
}

if ($RunOnly) {
    Start-Services
    exit 0
}

# Default: Build and Run
Write-Host "Building and running all services..." -ForegroundColor Yellow
Build-Services
Start-Services

Write-Host "`n=== Usage ===" -ForegroundColor Green
Write-Host "To view logs: .\start-all.ps1 -Logs" -ForegroundColor Cyan
Write-Host "To stop services: .\start-all.ps1 -Stop" -ForegroundColor Cyan
Write-Host "To clean environment: .\start-all.ps1 -Clean" -ForegroundColor Cyan
Write-Host "To build only: .\start-all.ps1 -BuildOnly" -ForegroundColor Cyan
Write-Host "To run only: .\start-all.ps1 -RunOnly" -ForegroundColor Cyan
