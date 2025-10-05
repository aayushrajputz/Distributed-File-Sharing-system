# Simple script to start all services locally
# Run this after infrastructure is started

Write-Host "Starting all services..." -ForegroundColor Cyan
Write-Host ""

$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path

# Set environment variables
$env:MONGO_URI = "mongodb://localhost:27017"
$env:MONGO_DATABASE = "file_sharing"
$env:REDIS_ADDR = "localhost:6379"
$env:MINIO_ENDPOINT = "localhost:9000"
$env:MINIO_ACCESS_KEY = "minioadmin"
$env:MINIO_SECRET_KEY = "minioadmin"
$env:MINIO_USE_SSL = "false"
$env:KAFKA_BROKERS = "localhost:9092"
$env:CASSANDRA_HOSTS = "localhost"
$env:JWT_SECRET = "your-super-secret-key-change-in-production"
$env:AUTH_SERVICE_ADDR = "localhost:50051"
$env:FILE_SERVICE_ADDR = "localhost:50052"
$env:NOTIFICATION_SERVICE_ADDR = "localhost:50053"
$env:BILLING_SERVICE_ADDR = "localhost:50054"

# Start Auth Service
Write-Host "1. Starting Auth Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList @(
    "-NoExit"
    "-Command"
    "Set-Location '$SCRIPT_DIR\services\auth-service'; .\auth-service.exe"
)
Start-Sleep -Seconds 2

# Start File Service
Write-Host "2. Starting File Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList @(
    "-NoExit"
    "-Command"
    "Set-Location '$SCRIPT_DIR\services\file-service'; .\file-service.exe"
)
Start-Sleep -Seconds 2

# Start Notification Service
Write-Host "3. Starting Notification Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList @(
    "-NoExit"
    "-Command"
    "Set-Location '$SCRIPT_DIR\services\notification-service'; .\notification-service.exe"
)
Start-Sleep -Seconds 2

# Start Billing Service
Write-Host "4. Starting Billing Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList @(
    "-NoExit"
    "-Command"
    "Set-Location '$SCRIPT_DIR\services\billing-service'; .\billing-service.exe"
)
Start-Sleep -Seconds 2

# Start Share Tracker
Write-Host "5. Starting Share Tracker..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList @(
    "-NoExit"
    "-Command"
    "Set-Location '$SCRIPT_DIR\services\share-tracker'; .\share-tracker.exe"
)
Start-Sleep -Seconds 5

# Start API Gateway
Write-Host "6. Starting API Gateway..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList @(
    "-NoExit"
    "-Command"
    "Set-Location '$SCRIPT_DIR\services\api-gateway'; .\api-gateway.exe"
)
Start-Sleep -Seconds 3

# Start Frontend
Write-Host "7. Starting Frontend..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList @(
    "-NoExit"
    "-Command"
    "Set-Location '$SCRIPT_DIR\frontend'; npm run dev"
)

Write-Host ""
Write-Host "All services started!" -ForegroundColor Green
Write-Host "Frontend: http://localhost:3000" -ForegroundColor Cyan
Write-Host "API Gateway: http://localhost:8080" -ForegroundColor Cyan
Write-Host ""
Write-Host "Press any key to exit this window..." -ForegroundColor Gray
$null = $Host.UI.RawUI.ReadKey("NoEcho,IncludeKeyDown")

