#!/bin/bash

# Script to generate protobuf code for billing service

set -e

echo "Generating protobuf code for billing service..."

# Navigate to project root
cd "$(dirname "$0")/.."

# Create output directory
mkdir -p services/billing-service/pkg/pb/billing/v1

# Generate protobuf code
protoc --proto_path=proto \
  --go_out=services/billing-service/pkg/pb \
  --go_opt=paths=source_relative \
  --go-grpc_out=services/billing-service/pkg/pb \
  --go-grpc_opt=paths=source_relative \
  --grpc-gateway_out=services/billing-service/pkg/pb \
  --grpc-gateway_opt=paths=source_relative \
  --grpc-gateway_opt=generate_unbound_methods=true \
  proto/billing/v1/billing.proto

echo "✅ Protobuf code generated successfully!"
echo ""
echo "Next steps:"
echo "1. cd services/billing-service"
echo "2. go mod tidy"
echo "3. go build ./cmd/server"
echo "4. docker-compose build billing-service"

