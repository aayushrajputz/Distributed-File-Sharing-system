Write-Host "=== Quick Start Script ===" -ForegroundColor Green

# Clean up old containers
Write-Host "Cleaning up old containers..." -ForegroundColor Cyan
docker-compose down -v

# Start everything
Write-Host "Starting all services..." -ForegroundColor Cyan
docker-compose up --build -d

Write-Host "`n=== Services Starting ===" -ForegroundColor Green
Write-Host "Frontend: http://localhost:3000"
Write-Host "API Gateway: http://localhost:8080"
Write-Host "Billing Service: http://localhost:8086"
Write-Host "`nUse 'docker-compose logs -f' to view logs"
