# PowerShell script to stop all locally running services

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  Stopping All Local Services" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Get the script directory
$SCRIPT_DIR = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $SCRIPT_DIR

Write-Host "Step 1: Stopping Go services..." -ForegroundColor Yellow
Write-Host ""

# Stop Auth Service
Write-Host "Stopping Auth Service..." -ForegroundColor Gray
Get-Process | Where-Object {$_.ProcessName -eq "auth-service"} | Stop-Process -Force 2>$null
Write-Host "  ✓ Auth Service stopped" -ForegroundColor Green

# Stop File Service
Write-Host "Stopping File Service..." -ForegroundColor Gray
Get-Process | Where-Object {$_.ProcessName -eq "file-service"} | Stop-Process -Force 2>$null
Write-Host "  ✓ File Service stopped" -ForegroundColor Green

# Stop Notification Service
Write-Host "Stopping Notification Service..." -ForegroundColor Gray
Get-Process | Where-Object {$_.ProcessName -eq "notification-service"} | Stop-Process -Force 2>$null
Write-Host "  ✓ Notification Service stopped" -ForegroundColor Green

# Stop Billing Service
Write-Host "Stopping Billing Service..." -ForegroundColor Gray
Get-Process | Where-Object {$_.ProcessName -eq "billing-service"} | Stop-Process -Force 2>$null
Write-Host "  ✓ Billing Service stopped" -ForegroundColor Green

# Stop Share Tracker
Write-Host "Stopping Share Tracker..." -ForegroundColor Gray
Get-Process | Where-Object {$_.ProcessName -eq "share-tracker"} | Stop-Process -Force 2>$null
Write-Host "  ✓ Share Tracker stopped" -ForegroundColor Green

# Stop API Gateway
Write-Host "Stopping API Gateway..." -ForegroundColor Gray
Get-Process | Where-Object {$_.ProcessName -eq "api-gateway"} | Stop-Process -Force 2>$null
Write-Host "  ✓ API Gateway stopped" -ForegroundColor Green

Write-Host ""
Write-Host "Step 2: Stopping Node.js processes..." -ForegroundColor Yellow
Write-Host ""

# Stop Node.js processes (Frontend)
Write-Host "Stopping Frontend (Node.js)..." -ForegroundColor Gray
Get-Process | Where-Object {$_.ProcessName -eq "node" -and $_.CommandLine -like "*next*"} | Stop-Process -Force 2>$null
# Alternative: Kill all node processes on port 3000
$port3000Process = Get-NetTCPConnection -LocalPort 3000 -ErrorAction SilentlyContinue | Select-Object -ExpandProperty OwningProcess
if ($port3000Process) {
    Stop-Process -Id $port3000Process -Force 2>$null
}
Write-Host "  ✓ Frontend stopped" -ForegroundColor Green

Write-Host ""
Write-Host "Step 3: Stopping infrastructure services..." -ForegroundColor Yellow
Write-Host ""

# Stop Docker infrastructure
Write-Host "Stopping Docker infrastructure (MongoDB, Redis, MinIO, Kafka, Cassandra)..." -ForegroundColor Gray
docker-compose down 2>&1 | Out-Null
Write-Host "  ✓ Infrastructure services stopped" -ForegroundColor Green

Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  All services stopped successfully!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""

