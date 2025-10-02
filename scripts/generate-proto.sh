#!/bin/bash

# Script to generate Go code from proto files

set -e

echo "Generating gRPC code from proto files..."

# Create output directories
mkdir -p services/auth-service/pkg/pb/auth/v1
mkdir -p services/file-service/pkg/pb/file/v1
mkdir -p services/notification-service/pkg/pb/notification/v1

# Install required tools if not present
echo "Checking for required tools..."
command -v protoc >/dev/null 2>&1 || { echo "protoc is not installed. Please install Protocol Buffer Compiler."; exit 1; }

# Generate Auth Service proto
echo "Generating Auth Service proto..."
protoc -I proto \
  -I third_party/googleapis \
  --go_out=services/auth-service/pkg/pb \
  --go_opt=paths=source_relative \
  --go-grpc_out=services/auth-service/pkg/pb \
  --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=services/auth-service/pkg/pb \
  --grpc-gateway_opt=paths=source_relative \
  --grpc-gateway_opt=generate_unbound_methods=true \
  proto/auth/v1/auth.proto

# Generate File Service proto
echo "Generating File Service proto..."
protoc -I proto \
  -I third_party/googleapis \
  --go_out=services/file-service/pkg/pb \
  --go_opt=paths=source_relative \
  --go-grpc_out=services/file-service/pkg/pb \
  --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=services/file-service/pkg/pb \
  --grpc-gateway_opt=paths=source_relative \
  --grpc-gateway_opt=generate_unbound_methods=true \
  proto/file/v1/file.proto

# Generate Notification Service proto
echo "Generating Notification Service proto..."
protoc -I proto \
  -I third_party/googleapis \
  --go_out=services/notification-service/pkg/pb \
  --go_opt=paths=source_relative \
  --go-grpc_out=services/notification-service/pkg/pb \
  --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=services/notification-service/pkg/pb \
  --grpc-gateway_opt=paths=source_relative \
  --grpc-gateway_opt=generate_unbound_methods=true \
  proto/notification/v1/notification.proto

echo "Proto generation complete!"

