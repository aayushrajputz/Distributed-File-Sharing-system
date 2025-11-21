# START ALL SERVICES LOCALLY
# This script will open each service in a new window

Write-Host "Starting all services..." -ForegroundColor Green

# Start each service in a new window
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\start-billing.bat"
Start-Sleep -Seconds 2

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\start-auth.bat"
Start-Sleep -Seconds 2

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\start-file.bat"
Start-Sleep -Seconds 2

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\start-notification.bat"
Start-Sleep -Seconds 2

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\start-gateway.bat"
Start-Sleep -Seconds 2

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\start-share-tracker.bat"
Start-Sleep -Seconds 2

Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD'; .\start-frontend.bat"

Write-Host "`nAll services are starting in separate windows!" -ForegroundColor Green
Write-Host "`nService URLs:" -ForegroundColor Cyan
Write-Host "  Frontend: http://localhost:3000"
Write-Host "  API Gateway: http://localhost:8080"
Write-Host "  Auth Service: http://localhost:8081"
Write-Host "  File Service: http://localhost:8082"
Write-Host "  Notification Service: http://localhost:8084"
Write-Host "  Billing Service: http://localhost:8086"
Write-Host "  Share Tracker: (Kafka consumer - no HTTP port)"
