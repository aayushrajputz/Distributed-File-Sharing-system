@echo off
echo Starting Auth Service...
set MONGO_URI=mongodb://localhost:27017
set MONGO_DATABASE=file_sharing
set REDIS_ADDR=localhost:6379
set JWT_SECRET=your-super-secret-key
set AUTH_SERVICE_PORT=8081
set AUTH_GRPC_PORT=50051

cd /d "%~dp0services\auth-service"
go run cmd/server/main.go
pause
