# Quick Start - Running Services Locally

This guide shows you how to quickly run all services locally for development.

## Prerequisites

- **Go** 1.19+ installed
- **Node.js** 16+ and npm installed
- **Docker Desktop** running

## Step 1: Start Infrastructure

Start MongoDB, Redis, MinIO, Kafka, and Cassandra with Docker:

```powershell
docker-compose up -d mongodb redis minio kafka zookeeper cassandra
```

Wait 30 seconds for services to be healthy.

## Step 2: Build All Services

```powershell
# Auth Service
cd services\auth-service
go build -o auth-service.exe cmd/server/main.go
cd ..\..

# File Service
cd services\file-service
go build -o file-service.exe cmd/server/main.go
cd ..\..

# Notification Service
cd services\notification-service
go build -o notification-service.exe cmd/server/main.go
cd ..\..

# Billing Service
cd services\billing-service
go build -o billing-service.exe cmd/server/main.go
cd ..\..

# Share Tracker
cd services\share-tracker
go build -o share-tracker.exe main.go
cd ..\..

# API Gateway
cd services\api-gateway
go build -o api-gateway.exe cmd/server/main.go
cd ..\..
```

## Step 3: Start All Services

Run the simple start script:

```powershell
.\start-services-simple.ps1
```

This will open 7 terminal windows:
1. Auth Service (port 50051)
2. File Service (port 50052)
3. Notification Service (port 50053)
4. Billing Service (port 50054)
5. Share Tracker
6. API Gateway (port 8080)
7. Frontend (port 3000)

## Step 4: Access the Application

Open your browser and go to:
- **Frontend**: http://localhost:3000
- **API Gateway**: http://localhost:8080

## Stopping Services

### Stop All Services
```powershell
.\stop-local.ps1
```

### Or Stop Manually
- Close each terminal window (Ctrl+C then close)
- Stop infrastructure:
  ```powershell
  docker-compose down
  ```

## Troubleshooting

### Port Already in Use

```powershell
# Find what's using the port
netstat -ano | findstr :PORT_NUMBER

# Kill the process
taskkill /F /PID PROCESS_ID
```

### Service Won't Start

1. Check if infrastructure is running:
   ```powershell
   docker ps
   ```

2. Make sure you built the service first

3. Check the terminal window for error messages

### Frontend Won't Start

```powershell
cd frontend
npm install
npm run dev
```

## Manual Service Start (Alternative)

If the script doesn't work, you can start each service manually in separate PowerShell windows:

### Terminal 1 - Auth Service
```powershell
cd services\auth-service
.\auth-service.exe
```

### Terminal 2 - File Service
```powershell
cd services\file-service
.\file-service.exe
```

### Terminal 3 - Notification Service
```powershell
cd services\notification-service
.\notification-service.exe
```

### Terminal 4 - Billing Service
```powershell
cd services\billing-service
.\billing-service.exe
```

### Terminal 5 - Share Tracker
```powershell
cd services\share-tracker
.\share-tracker.exe
```

### Terminal 6 - API Gateway
```powershell
cd services\api-gateway
.\api-gateway.exe
```

### Terminal 7 - Frontend
```powershell
cd frontend
npm run dev
```

## Environment Variables

The services use these default values for local development:

- `MONGO_URI=mongodb://localhost:27017`
- `REDIS_ADDR=localhost:6379`
- `MINIO_ENDPOINT=localhost:9000`
- `KAFKA_BROKERS=localhost:9092`
- `CASSANDRA_HOSTS=localhost`
- `AUTH_SERVICE_ADDR=localhost:50051`
- `FILE_SERVICE_ADDR=localhost:50052`
- `NOTIFICATION_SERVICE_ADDR=localhost:50053`
- `BILLING_SERVICE_ADDR=localhost:50054`

## Benefits of Local Development

✅ Faster iteration - no Docker image rebuilds
✅ Better debugging - direct access to logs
✅ Hot reload for frontend
✅ Less resource usage
✅ Easier to test individual services

## Next Steps

- See [LOCAL_DEVELOPMENT.md](LOCAL_DEVELOPMENT.md) for detailed documentation
- See [README.md](README.md) for Docker-based setup

