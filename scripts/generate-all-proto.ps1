$ErrorActionPreference = "Stop"

Write-Host "Generating gRPC code from proto files..." -ForegroundColor Cyan

# Create output directories
New-Item -ItemType Directory -Force -Path "..\services\auth-service\pkg\pb\auth\v1" | Out-Null
New-Item -ItemType Directory -Force -Path "..\services\file-service\pkg\pb\file\v1" | Out-Null
New-Item -ItemType Directory -Force -Path "..\services\notification-service\pkg\pb\notification\v1" | Out-Null
New-Item -ItemType Directory -Force -Path "..\services\billing-service\pkg\pb\billing\v1" | Out-Null

# Check for protoc
$ProtocPath = "..\protoc_extracted\bin\protoc.exe"
if (-not (Test-Path $ProtocPath)) {
    Write-Host "protoc not found at $ProtocPath. Please run the setup or ensure protoc is extracted." -ForegroundColor Red
    exit 1
}

# Generate Auth Service proto
Write-Host "Generating Auth Service proto..."
& $ProtocPath -I ..\proto `
  -I ..\third_party\googleapis `
  --go_out=..\services\auth-service\pkg\pb `
  --go_opt=paths=source_relative `
  --go-grpc_out=..\services\auth-service\pkg\pb `
  --go-grpc_opt=paths=source_relative `
  --grpc-gateway_out=..\services\auth-service\pkg\pb `
  --grpc-gateway_opt=paths=source_relative `
  --grpc-gateway_opt=generate_unbound_methods=true `
  ..\proto\auth\v1\auth.proto

# Generate File Service proto
Write-Host "Generating File Service proto..."
& $ProtocPath -I ..\proto `
  -I ..\third_party\googleapis `
  --go_out=..\services\file-service\pkg\pb `
  --go_opt=paths=source_relative `
  --go-grpc_out=..\services\file-service\pkg\pb `
  --go-grpc_opt=paths=source_relative `
  --grpc-gateway_out=..\services\file-service\pkg\pb `
  --grpc-gateway_opt=paths=source_relative `
  --grpc-gateway_opt=generate_unbound_methods=true `
  ..\proto\file\v1\file.proto

Write-Host "Generating File Service Private Folder proto..."
& $ProtocPath -I ..\proto `
  -I ..\third_party\googleapis `
  --go_out=..\services\file-service\pkg\pb `
  --go_opt=paths=source_relative `
  --go-grpc_out=..\services\file-service\pkg\pb `
  --go-grpc_opt=paths=source_relative `
  --grpc-gateway_out=..\services\file-service\pkg\pb `
  --grpc-gateway_opt=paths=source_relative `
  --grpc-gateway_opt=generate_unbound_methods=true `
  ..\proto\file\v1\private_folder.proto

# Generate Notification Service proto
Write-Host "Generating Notification Service proto..."
& $ProtocPath -I ..\proto `
  -I ..\third_party\googleapis `
  --go_out=..\services\notification-service\pkg\pb `
  --go_opt=paths=source_relative `
  --go-grpc_out=..\services\notification-service\pkg\pb `
  --go-grpc_opt=paths=source_relative `
  --grpc-gateway_out=..\services\notification-service\pkg\pb `
  --grpc-gateway_opt=paths=source_relative `
  --grpc-gateway_opt=generate_unbound_methods=true `
  ..\proto\notification\v1\notification.proto

# Generate Billing Service proto
Write-Host "Generating Billing Service proto..."
& $ProtocPath -I ..\proto `
  -I ..\third_party\googleapis `
  --go_out=..\services\billing-service\pkg\pb `
  --go_opt=paths=source_relative `
  --go-grpc_out=..\services\billing-service\pkg\pb `
  --go-grpc_opt=paths=source_relative `
  --grpc-gateway_out=..\services\billing-service\pkg\pb `
  --grpc-gateway_opt=paths=source_relative `
  --grpc-gateway_opt=generate_unbound_methods=true `
  ..\proto\billing\v1\billing.proto

Write-Host "Proto generation complete!" -ForegroundColor Green
