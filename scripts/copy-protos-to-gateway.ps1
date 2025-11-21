Write-Host "Copying proto files to api-gateway..." -ForegroundColor Cyan

# Create directories
New-Item -ItemType Directory -Force -Path "services\api-gateway\pkg\pb\auth\v1" | Out-Null
New-Item -ItemType Directory -Force -Path "services\api-gateway\pkg\pb\file\v1" | Out-Null
New-Item -ItemType Directory -Force -Path "services\api-gateway\pkg\pb\notification\v1" | Out-Null
New-Item -ItemType Directory -Force -Path "services\api-gateway\pkg\pb\billing\v1" | Out-Null

# Copy proto files
Copy-Item -Path "services\auth-service\pkg\pb\auth\v1\*" -Destination "services\api-gateway\pkg\pb\auth\v1\" -Force
Copy-Item -Path "services\file-service\pkg\pb\file\v1\*" -Destination "services\api-gateway\pkg\pb\file\v1\" -Force
Copy-Item -Path "services\notification-service\pkg\pb\notification\v1\*" -Destination "services\api-gateway\pkg\pb\notification\v1\" -Force
Copy-Item -Path "services\billing-service\pkg\pb\billing\v1\*" -Destination "services\api-gateway\pkg\pb\billing\v1\" -Force

Write-Host "Proto files copied successfully!" -ForegroundColor Green
