# Local Development Guide

This guide explains how to run all services locally (outside of Docker) for development purposes.

## Prerequisites

Before running services locally, ensure you have the following installed:

1. **Go** (version 1.19 or higher)
   - Download from: https://golang.org/dl/
   - Verify installation: `go version`

2. **Node.js** (version 16 or higher) and npm
   - Download from: https://nodejs.org/
   - Verify installation: `node --version` and `npm --version`

3. **Docker Desktop** (for infrastructure services)
   - Download from: https://www.docker.com/products/docker-desktop
   - Required for MongoDB, Redis, MinIO, Kafka, Zookeeper, and Cassandra

## Quick Start

### Starting All Services

Run the following command from the project root directory:

```powershell
.\run-local.ps1
```

This script will:
1. Check prerequisites (Go, Node.js)
2. Start infrastructure services with Docker (MongoDB, Redis, MinIO, Kafka, Cassandra)
3. Build all Go services
4. Start all backend services in separate terminal windows
5. Start the frontend in a separate terminal window

### Stopping All Services

Run the following command from the project root directory:

```powershell
.\stop-local.ps1
```

This script will:
1. Stop all Go service processes
2. Stop the Node.js frontend process
3. Stop Docker infrastructure services

## Service URLs

Once all services are running, you can access them at:

### User-Facing Services
- **Frontend**: http://localhost:3000
- **API Gateway**: http://localhost:8080

### Backend Services (gRPC)
- **Auth Service**: localhost:50051
- **File Service**: localhost:50052
- **Notification Service**: localhost:50053
- **Billing Service**: localhost:50054

### Infrastructure Services
- **MongoDB**: localhost:27017
- **Redis**: localhost:6379
- **MinIO Console**: http://localhost:9000 (credentials: minioadmin/minioadmin)
- **Kafka**: localhost:9092
- **Cassandra**: localhost:9042

## Manual Service Management

If you prefer to start services manually or need to restart individual services:

### Infrastructure Services

Start only infrastructure:
```powershell
docker-compose up -d mongodb redis minio kafka zookeeper cassandra
```

Stop infrastructure:
```powershell
docker-compose down
```

### Backend Services

Each service can be built and run individually:

#### Auth Service
```powershell
cd services\auth-service
go build -o auth-service.exe cmd/server/main.go
.\auth-service.exe
```

#### File Service
```powershell
cd services\file-service
go build -o file-service.exe cmd/server/main.go
.\file-service.exe
```

#### Notification Service
```powershell
cd services\notification-service
go build -o notification-service.exe cmd/server/main.go
.\notification-service.exe
```

#### Billing Service
```powershell
cd services\billing-service
go build -o billing-service.exe cmd/server/main.go
.\billing-service.exe
```

#### Share Tracker
```powershell
cd services\share-tracker
go build -o share-tracker.exe main.go
.\share-tracker.exe
```

#### API Gateway
```powershell
cd services\api-gateway
go build -o api-gateway.exe cmd/server/main.go
.\api-gateway.exe
```

### Frontend

```powershell
cd frontend
npm install  # Only needed first time or after package.json changes
npm run dev
```

## Environment Variables

The `run-local.ps1` script sets the following environment variables automatically:

### Database & Cache
- `MONGO_URI=mongodb://localhost:27017`
- `MONGO_DATABASE=file_sharing`
- `REDIS_ADDR=localhost:6379`
- `REDIS_PASSWORD=`
- `REDIS_DB=0`

### Object Storage
- `MINIO_ENDPOINT=localhost:9000`
- `MINIO_ACCESS_KEY=minioadmin`
- `MINIO_SECRET_KEY=minioadmin`
- `MINIO_USE_SSL=false`
- `MINIO_BUCKET=file-sharing`

### Message Queue
- `KAFKA_BROKERS=localhost:9092`

### Analytics Database
- `CASSANDRA_HOSTS=localhost`
- `CASSANDRA_KEYSPACE=file_service`

### Security
- `JWT_SECRET=your-super-secret-key-change-in-production`

### Service Ports
- `AUTH_SERVICE_PORT=50051`
- `FILE_SERVICE_PORT=50052`
- `NOTIFICATION_SERVICE_PORT=50053`
- `BILLING_SERVICE_PORT=50054`
- `API_GATEWAY_PORT=8080`
- `FRONTEND_PORT=3000`

### Service Addresses
- `AUTH_SERVICE_ADDR=localhost:50051`
- `FILE_SERVICE_ADDR=localhost:50052`
- `NOTIFICATION_SERVICE_ADDR=localhost:50053`
- `BILLING_SERVICE_ADDR=localhost:50054`

## Troubleshooting

### Port Already in Use

If you get "port already in use" errors:

1. Check what's using the port:
   ```powershell
   netstat -ano | findstr :PORT_NUMBER
   ```

2. Kill the process:
   ```powershell
   taskkill /F /PID PROCESS_ID
   ```

3. Or use the stop script:
   ```powershell
   .\stop-local.ps1
   ```

### Service Won't Start

1. Check if infrastructure services are running:
   ```powershell
   docker ps
   ```

2. Check service logs in the terminal window where the service is running

3. Verify environment variables are set correctly

### Frontend Build Errors

If the frontend fails to start:

1. Delete node_modules and reinstall:
   ```powershell
   cd frontend
   Remove-Item -Recurse -Force node_modules
   npm install
   ```

2. Clear Next.js cache:
   ```powershell
   Remove-Item -Recurse -Force .next
   npm run dev
   ```

### Database Connection Errors

1. Ensure Docker infrastructure is running:
   ```powershell
   docker-compose ps
   ```

2. Wait for services to be healthy (30-60 seconds after starting)

3. Check Docker logs:
   ```powershell
   docker logs mongodb
   docker logs redis
   docker logs cassandra
   ```

## Development Workflow

### Making Code Changes

1. **Backend Services (Go)**:
   - Make your changes
   - Stop the service (Ctrl+C in its terminal)
   - Rebuild: `go build -o service-name.exe cmd/server/main.go`
   - Restart the service: `.\service-name.exe`

2. **Frontend (Next.js)**:
   - Make your changes
   - Hot reload will automatically apply changes
   - If hot reload doesn't work, restart with Ctrl+C and `npm run dev`

### Running Tests

```powershell
# Run tests for a specific service
cd services\SERVICE_NAME
go test ./...

# Run frontend tests
cd frontend
npm test
```

### Viewing Logs

Each service runs in its own terminal window, so logs are visible in real-time. You can also:

- Redirect logs to files when starting services manually
- Use the Docker logs command for infrastructure services
- Check the browser console for frontend errors

## Advantages of Local Development

- **Faster iteration**: No need to rebuild Docker images
- **Better debugging**: Direct access to service logs and debugger
- **Resource efficient**: Only infrastructure runs in Docker
- **Hot reload**: Frontend changes apply immediately
- **Easy testing**: Can run individual services or tests

## Switching Back to Docker

To run everything in Docker again:

1. Stop local services:
   ```powershell
   .\stop-local.ps1
   ```

2. Start all services with Docker:
   ```powershell
   docker-compose up -d
   ```

## Additional Resources

- [Main README](README.md) - Project overview and Docker setup
- [API Documentation](docs/api.md) - API endpoints and usage
- [Architecture](docs/architecture.md) - System architecture overview

