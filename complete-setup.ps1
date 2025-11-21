Write-Host "=== Complete Setup and Start ===" -ForegroundColor Green

# Step 1: Generate Proto Files
Write-Host "`n1. Generating proto files..." -ForegroundColor Cyan
cd scripts
..\scripts\generate-all-proto.ps1
cd ..

# Step 2: Copy protos to API Gateway
Write-Host "`n2. Copying protos to API Gateway..." -ForegroundColor Cyan
.\scripts\copy-protos-to-gateway.ps1

# Step 3: Start Infrastructure
Write-Host "`n3. Starting infrastructure services..." -ForegroundColor Cyan
docker-compose up -d mongodb redis minio zookeeper kafka cassandra

Write-Host "Waiting 20s for infrastructure..." -ForegroundColor Yellow
Start-Sleep -Seconds 20

# Step 4: Build and start services one by one
Write-Host "`n4. Building and starting application services..." -ForegroundColor Cyan

$services = @("auth-service", "file-service", "notification-service", "billing-service", "api-gateway", "share-tracker", "frontend")

foreach ($service in $services) {
    Write-Host "Building $service..." -ForegroundColor Yellow
    docker-compose build $service 2>$null
    if ($LASTEXITCODE -eq 0) {
        Write-Host "✓ $service built successfully" -ForegroundColor Green
        docker-compose up -d $service
    } else {
        Write-Host "✗ $service build failed, skipping..." -ForegroundColor Red
    }
}

Write-Host "`n=== Setup Complete ===" -ForegroundColor Green
Write-Host "Check running services with: docker-compose ps"
Write-Host "View logs with: docker-compose logs -f"
Write-Host "`nService URLs:"
Write-Host "Frontend: http://localhost:3000"
Write-Host "API Gateway: http://localhost:8080"
Write-Host "Billing Service: http://localhost:8086"
