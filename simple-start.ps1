Write-Host "=== Simple Start Script ===" -ForegroundColor Green

# 1. Generate Protos
Write-Host "1. Generating Protos..." -ForegroundColor Cyan
Set-Location scripts
.\generate-all-proto.ps1
if ($LASTEXITCODE -ne 0) { Write-Host "Proto generation failed"; exit 1 }
Set-Location ..

# 2. Build Images
Write-Host "2. Building Docker Images..." -ForegroundColor Cyan
docker-compose build --no-cache --parallel
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed"; exit 1 }

# 3. Start Infrastructure
Write-Host "3. Starting Infrastructure..." -ForegroundColor Cyan
docker-compose up -d mongodb redis minio zookeeper kafka cassandra

Write-Host "Waiting 30s for infrastructure..." -ForegroundColor Yellow
Start-Sleep -Seconds 30

# 4. Initialize Cassandra
Write-Host "4. Initializing Cassandra..." -ForegroundColor Cyan
docker-compose up cassandra-init

# 5. Start Application Services
Write-Host "5. Starting Application Services..." -ForegroundColor Cyan
docker-compose up -d auth-service file-service notification-service billing-service api-gateway share-tracker frontend

Write-Host "=== All Services Started ===" -ForegroundColor Green
Write-Host "Frontend: http://localhost:3000"
Write-Host "API Gateway: http://localhost:8080"
