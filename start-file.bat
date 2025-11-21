@echo off
echo Starting File Service...
set MONGO_URI=mongodb://localhost:27017
set MONGO_DATABASE=file_sharing
set REDIS_ADDR=localhost:6379
set MINIO_ENDPOINT=localhost:9000
set MINIO_ACCESS_KEY=minioadmin
set MINIO_SECRET_KEY=minioadmin
set MINIO_USE_SSL=false
set MINIO_BUCKET=file-sharing
set KAFKA_BROKERS=localhost:9092
set CASSANDRA_HOSTS=localhost
set JWT_SECRET=your-super-secret-key
set FILE_SERVICE_PORT=8082
set FILE_GRPC_PORT=50052

cd /d "%~dp0services\file-service"
go run cmd/server/main.go
pause
