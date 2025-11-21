@echo off
echo Starting Notification Service...
set MONGO_URI=mongodb://localhost:27017
set MONGO_DATABASE=file_sharing
set KAFKA_BROKERS=localhost:9092
set NOTIFICATION_SERVICE_PORT=8084
set NOTIFICATION_GRPC_PORT=50053
set NOTIFICATION_WEBSOCKET_PORT=8085
set NOTIFICATION_METRICS_PORT=9095

cd /d "%~dp0services\notification-service"
go run cmd/server/main.go
pause
