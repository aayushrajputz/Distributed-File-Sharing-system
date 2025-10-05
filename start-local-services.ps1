# Start Local Services Script
# This script starts all backend services locally with proper environment variables

Write-Host "Starting infrastructure services..." -ForegroundColor Green

# Wait for infrastructure to be ready
Start-Sleep -Seconds 5

Write-Host "Starting Auth Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd 'C:\Users\aayus\OneDrive\Desktop\New folder\distributed-file-sharing-platform\services\auth-service'; `$env:MONGO_URI='mongodb://localhost:27017'; `$env:MONGO_DATABASE='file_sharing'; `$env:REDIS_ADDR='localhost:6379'; `$env:JWT_SECRET='your-super-secret-key-change-in-production'; `$env:AUTH_SERVICE_PORT='8081'; `$env:AUTH_GRPC_PORT='50051'; .\auth-service.exe"

Start-Sleep -Seconds 3

Write-Host "Starting File Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd 'C:\Users\aayus\OneDrive\Desktop\New folder\distributed-file-sharing-platform\services\file-service'; `$env:MONGO_URI='mongodb://localhost:27017'; `$env:MONGO_DATABASE='file_sharing'; `$env:MINIO_ENDPOINT='localhost:9000'; `$env:MINIO_ACCESS_KEY='minioadmin'; `$env:MINIO_SECRET_KEY='minioadmin'; `$env:MINIO_USE_SSL='false'; `$env:KAFKA_BROKERS='localhost:9092'; `$env:REDIS_ADDR='localhost:6379'; `$env:AUTH_SERVICE_GRPC='localhost:50051'; `$env:JWT_SECRET='your-super-secret-key-change-in-production'; `$env:FILE_SERVICE_PORT='8082'; `$env:FILE_GRPC_PORT='50052'; .\file-service.exe"

Start-Sleep -Seconds 3

Write-Host "Starting Notification Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd 'C:\Users\aayus\OneDrive\Desktop\New folder\distributed-file-sharing-platform\services\notification-service'; `$env:MONGO_URI='mongodb://localhost:27017'; `$env:MONGO_DATABASE='file_sharing'; `$env:REDIS_ADDR='localhost:6379'; `$env:JWT_SECRET='your-super-secret-key-change-in-production'; `$env:NOTIFICATION_SERVICE_PORT='8084'; `$env:NOTIFICATION_GRPC_PORT='50054'; .\notification-service.exe"

Start-Sleep -Seconds 3

Write-Host "Starting Billing Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd 'C:\Users\aayus\OneDrive\Desktop\New folder\distributed-file-sharing-platform\services\billing-service'; `$env:MONGO_URI='mongodb://localhost:27017'; `$env:MONGO_DATABASE='file_sharing'; `$env:REDIS_ADDR='localhost:6379'; `$env:JWT_SECRET='your-super-secret-key-change-in-production'; `$env:BILLING_SERVICE_PORT='8086'; `$env:BILLING_GRPC_PORT='50056'; .\billing-service.exe"

Start-Sleep -Seconds 3

Write-Host "Starting Share Tracker Service..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd 'C:\Users\aayus\OneDrive\Desktop\New folder\distributed-file-sharing-platform\services\share-tracker'; `$env:MONGO_URI='mongodb://localhost:27017'; `$env:MONGO_DATABASE='file_sharing'; `$env:REDIS_ADDR='localhost:6379'; `$env:JWT_SECRET='your-super-secret-key-change-in-production'; `$env:SHARE_TRACKER_SERVICE_PORT='8087'; `$env:SHARE_TRACKER_GRPC_PORT='50057'; .\share-tracker.exe"

Start-Sleep -Seconds 3

Write-Host "Starting API Gateway..." -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd 'C:\Users\aayus\OneDrive\Desktop\New folder\distributed-file-sharing-platform\services\api-gateway'; `$env:MONGO_URI='mongodb://localhost:27017'; `$env:MONGO_DATABASE='file_sharing'; `$env:REDIS_ADDR='localhost:6379'; `$env:JWT_SECRET='your-super-secret-key-change-in-production'; `$env:API_GATEWAY_PORT='8080'; `$env:AUTH_SERVICE_GRPC='localhost:50051'; `$env:FILE_SERVICE_GRPC='localhost:50052'; `$env:NOTIFICATION_SERVICE_GRPC='localhost:50054'; `$env:BILLING_SERVICE_GRPC='localhost:50056'; `$env:SHARE_TRACKER_SERVICE_GRPC='localhost:50057'; .\api-gateway.exe"

Write-Host "All services started! Check the individual terminal windows for any errors." -ForegroundColor Green
Write-Host "Services should be available at:" -ForegroundColor Cyan
Write-Host "  - Auth Service: http://localhost:8081" -ForegroundColor White
Write-Host "  - File Service: http://localhost:8082" -ForegroundColor White
Write-Host "  - Notification Service: http://localhost:8084" -ForegroundColor White
Write-Host "  - Billing Service: http://localhost:8086" -ForegroundColor White
Write-Host "  - Share Tracker: http://localhost:8087" -ForegroundColor White
Write-Host "  - API Gateway: http://localhost:8080" -ForegroundColor White
