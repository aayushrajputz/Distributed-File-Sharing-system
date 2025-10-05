# PowerShell script to run all services locally
# This script starts all backend services and the frontend

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Distributed File Sharing Platform" -ForegroundColor Cyan
Write-Host "  Local Development Environment" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Set error action preference
$ErrorActionPreference = "Continue"

# Get the script directory
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $SCRIPT_DIR

# Environment variables for local development
$env:MONGO_URI = "mongodb://localhost:27017"
$env:MONGO_DATABASE = "file_sharing"
$env:REDIS_ADDR = "localhost:6379"
$env:REDIS_PASSWORD = ""
$env:REDIS_DB = "0"
$env:MINIO_ENDPOINT = "localhost:9000"
$env:MINIO_ACCESS_KEY = "minioadmin"
$env:MINIO_SECRET_KEY = "minioadmin"
$env:MINIO_USE_SSL = "false"
$env:MINIO_BUCKET = "file-sharing"
$env:KAFKA_BROKERS = "localhost:9092"
$env:CASSANDRA_HOSTS = "localhost"
$env:CASSANDRA_KEYSPACE = "file_service"
$env:JWT_SECRET = "your-super-secret-key-change-in-production"
$env:ENVIRONMENT = "development"
$env:LOG_LEVEL = "debug"

# Service ports
$env:AUTH_SERVICE_PORT = "50051"
$env:AUTH_SERVICE_HTTP_PORT = "8081"
$env:FILE_SERVICE_PORT = "50052"
$env:FILE_SERVICE_HTTP_PORT = "8082"
$env:NOTIFICATION_SERVICE_PORT = "50053"
$env:NOTIFICATION_SERVICE_HTTP_PORT = "8083"
$env:BILLING_SERVICE_PORT = "50054"
$env:BILLING_SERVICE_HTTP_PORT = "8086"
$env:API_GATEWAY_PORT = "8080"
$env:FRONTEND_PORT = "3000"

# gRPC service addresses
$env:AUTH_SERVICE_ADDR = "localhost:50051"
$env:FILE_SERVICE_ADDR = "localhost:50052"
$env:NOTIFICATION_SERVICE_ADDR = "localhost:50053"
$env:BILLING_SERVICE_ADDR = "localhost:50054"

Write-Host "Step 1: Checking prerequisites..." -ForegroundColor Yellow
Write-Host ""

# Check if Go is installed
Write-Host "Checking Go installation..." -ForegroundColor Gray
$goVersion = go version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Go is not installed. Please install Go from https://golang.org/dl/" -ForegroundColor Red
    exit 1
}
Write-Host "  ✓ Go is installed: $goVersion" -ForegroundColor Green

# Check if Node.js is installed
Write-Host "Checking Node.js installation..." -ForegroundColor Gray
$nodeVersion = node --version 2>$null
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Node.js is not installed. Please install Node.js from https://nodejs.org/" -ForegroundColor Red
    exit 1
}
Write-Host "  ✓ Node.js is installed: $nodeVersion" -ForegroundColor Green

Write-Host ""
Write-Host "Step 2: Starting infrastructure services with Docker..." -ForegroundColor Yellow
Write-Host ""

# Start only infrastructure services (MongoDB, Redis, MinIO, Kafka, Zookeeper, Cassandra)
Write-Host "Starting MongoDB, Redis, MinIO, Kafka, Zookeeper, and Cassandra..." -ForegroundColor Gray
docker-compose up -d mongodb redis minio kafka zookeeper cassandra 2>&1 | Out-Null

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to start infrastructure services" -ForegroundColor Red
    exit 1
}

Write-Host "  ✓ Infrastructure services starting..." -ForegroundColor Green
Write-Host "  Waiting for services to be healthy - 30 seconds..." -ForegroundColor Gray
Start-Sleep -Seconds 30

Write-Host ""
Write-Host "Step 3: Building Go services..." -ForegroundColor Yellow
Write-Host ""

# Build Auth Service
Write-Host "Building Auth Service..." -ForegroundColor Gray
Set-Location "services\auth-service"
go build -o auth-service.exe cmd/server/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to build Auth Service" -ForegroundColor Red
    Set-Location $SCRIPT_DIR
    exit 1
}
Write-Host "  ✓ Auth Service built successfully" -ForegroundColor Green
Set-Location $SCRIPT_DIR

# Build File Service
Write-Host "Building File Service..." -ForegroundColor Gray
Set-Location "services\file-service"
go build -o file-service.exe cmd/server/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to build File Service" -ForegroundColor Red
    Set-Location $SCRIPT_DIR
    exit 1
}
Write-Host "  ✓ File Service built successfully" -ForegroundColor Green
Set-Location $SCRIPT_DIR

# Build Notification Service
Write-Host "Building Notification Service..." -ForegroundColor Gray
Set-Location "services\notification-service"
go build -o notification-service.exe cmd/server/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to build Notification Service" -ForegroundColor Red
    Set-Location $SCRIPT_DIR
    exit 1
}
Write-Host "  ✓ Notification Service built successfully" -ForegroundColor Green
Set-Location $SCRIPT_DIR

# Build Billing Service
Write-Host "Building Billing Service..." -ForegroundColor Gray
Set-Location "services\billing-service"
go build -o billing-service.exe cmd/server/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to build Billing Service" -ForegroundColor Red
    Set-Location $SCRIPT_DIR
    exit 1
}
Write-Host "  ✓ Billing Service built successfully" -ForegroundColor Green
Set-Location $SCRIPT_DIR

# Build Share Tracker
Write-Host "Building Share Tracker..." -ForegroundColor Gray
Set-Location "services\share-tracker"
go build -o share-tracker.exe main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to build Share Tracker" -ForegroundColor Red
    Set-Location $SCRIPT_DIR
    exit 1
}
Write-Host "  ✓ Share Tracker built successfully" -ForegroundColor Green
Set-Location $SCRIPT_DIR

# Build API Gateway
Write-Host "Building API Gateway..." -ForegroundColor Gray
Set-Location "services\api-gateway"
go build -o api-gateway.exe cmd/server/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to build API Gateway" -ForegroundColor Red
    Set-Location $SCRIPT_DIR
    exit 1
}
Write-Host "  ✓ API Gateway built successfully" -ForegroundColor Green
Set-Location $SCRIPT_DIR

Write-Host ""
Write-Host "Step 4: Starting backend services..." -ForegroundColor Yellow
Write-Host ""

# Start Auth Service
Write-Host "Starting Auth Service on port 50051..." -ForegroundColor Gray
$authCmd = "Set-Location '$SCRIPT_DIR\services\auth-service'; .\auth-service.exe"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $authCmd
Start-Sleep -Seconds 2
Write-Host "  ✓ Auth Service started" -ForegroundColor Green

# Start File Service
Write-Host "Starting File Service on port 50052..." -ForegroundColor Gray
$fileCmd = "Set-Location '$SCRIPT_DIR\services\file-service'; .\file-service.exe"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $fileCmd
Start-Sleep -Seconds 2
Write-Host "  ✓ File Service started" -ForegroundColor Green

# Start Notification Service
Write-Host "Starting Notification Service on port 50053..." -ForegroundColor Gray
$notifCmd = "Set-Location '$SCRIPT_DIR\services\notification-service'; .\notification-service.exe"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $notifCmd
Start-Sleep -Seconds 2
Write-Host "  ✓ Notification Service started" -ForegroundColor Green

# Start Billing Service
Write-Host "Starting Billing Service on port 50054..." -ForegroundColor Gray
$billingCmd = "Set-Location '$SCRIPT_DIR\services\billing-service'; .\billing-service.exe"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $billingCmd
Start-Sleep -Seconds 2
Write-Host "  ✓ Billing Service started" -ForegroundColor Green

# Start Share Tracker
Write-Host "Starting Share Tracker..." -ForegroundColor Gray
$shareCmd = "Set-Location '$SCRIPT_DIR\services\share-tracker'; .\share-tracker.exe"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $shareCmd
Start-Sleep -Seconds 2
Write-Host "  ✓ Share Tracker started" -ForegroundColor Green

# Wait for services to initialize
Write-Host ""
Write-Host "Waiting for backend services to initialize - 10 seconds..." -ForegroundColor Gray
Start-Sleep -Seconds 10

# Start API Gateway
Write-Host ""
Write-Host "Starting API Gateway on port 8080..." -ForegroundColor Gray
$gatewayCmd = "Set-Location '$SCRIPT_DIR\services\api-gateway'; .\api-gateway.exe"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $gatewayCmd
Start-Sleep -Seconds 3
Write-Host "  ✓ API Gateway started" -ForegroundColor Green

Write-Host ""
Write-Host "Step 5: Starting frontend..." -ForegroundColor Yellow
Write-Host ""

# Start Frontend
Write-Host "Starting Frontend on port 3000..." -ForegroundColor Gray
$frontendCmd = "Set-Location '$SCRIPT_DIR\frontend'; npm run dev"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $frontendCmd
Start-Sleep -Seconds 3
Write-Host "  ✓ Frontend started" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  All services started successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "Service URLs:" -ForegroundColor Cyan
Write-Host "  Frontend:              http://localhost:3000" -ForegroundColor White
Write-Host "  API Gateway:           http://localhost:8080" -ForegroundColor White
Write-Host "  Auth Service:          localhost:50051 (gRPC)" -ForegroundColor White
Write-Host "  File Service:          localhost:50052 (gRPC)" -ForegroundColor White
Write-Host "  Notification Service:  localhost:50053 (gRPC)" -ForegroundColor White
Write-Host "  Billing Service:       localhost:50054 (gRPC)" -ForegroundColor White
Write-Host ""
Write-Host "Infrastructure:" -ForegroundColor Cyan
Write-Host "  MongoDB:               localhost:27017" -ForegroundColor White
Write-Host "  Redis:                 localhost:6379" -ForegroundColor White
Write-Host "  MinIO:                 localhost:9000" -ForegroundColor White
Write-Host "  Kafka:                 localhost:9092" -ForegroundColor White
Write-Host "  Cassandra:             localhost:9042" -ForegroundColor White
Write-Host ""
Write-Host "Press Ctrl+C in each terminal window to stop services" -ForegroundColor Yellow
Write-Host "To stop infrastructure: docker-compose down" -ForegroundColor Yellow
Write-Host ""

