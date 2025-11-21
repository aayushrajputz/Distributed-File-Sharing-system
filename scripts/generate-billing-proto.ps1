$ErrorActionPreference = "Stop"

Write-Host "Generating Billing Service proto..."

# Create output directory
New-Item -ItemType Directory -Force -Path "services/billing-service/pkg/pb/billing/v1" | Out-Null

# Generate Billing Service proto
.\protoc_extracted\bin\protoc.exe -I proto `
  -I third_party/googleapis `
  --go_out=services/billing-service/pkg/pb `
  --go_opt=paths=source_relative `
  --go-grpc_out=services/billing-service/pkg/pb `
  --go-grpc_opt=paths=source_relative `
  --grpc-gateway_out=services/billing-service/pkg/pb `
  --grpc-gateway_opt=paths=source_relative `
  --grpc-gateway_opt=generate_unbound_methods=true `
  proto/billing/v1/billing.proto

Write-Host "Proto generation complete!"
