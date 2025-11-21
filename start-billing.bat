@echo off
echo Starting Billing Service...
set MONGO_URI=mongodb://localhost:27017
set MONGO_DATABASE=file_sharing
set REDIS_ADDR=localhost:6379
set MINIO_ENDPOINT=localhost:9000
set MINIO_ACCESS_KEY=minioadmin
set MINIO_SECRET_KEY=minioadmin
set KAFKA_BROKERS=localhost:9092
set CASSANDRA_HOSTS=localhost
set JWT_SECRET=your-super-secret-key
set BILLING_SERVICE_PORT=8086
set BILLING_GRPC_PORT=50054
set FILE_SERVICE_GRPC=localhost:50052

cd /d "%~dp0services\billing-service"
go run cmd/server/main.go
pause
