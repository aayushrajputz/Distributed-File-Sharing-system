@echo off
echo Starting API Gateway...
set AUTH_SERVICE_ADDR=localhost:50051
set FILE_SERVICE_ADDR=localhost:50052
set NOTIFICATION_SERVICE_ADDR=localhost:50053
set BILLING_SERVICE_ADDR=localhost:50054
set JWT_SECRET=your-super-secret-key
set API_GATEWAY_PORT=8080
set NOTIFICATION_SERVICE_REST_URL=http://localhost:8084

cd /d "%~dp0services\api-gateway"
go run cmd/server/main.go
pause
