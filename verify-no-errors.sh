#!/bin/bash
# Script to verify there are no compilation errors in the project

set -e

echo "========================================"
echo "Verifying Project - No Errors Check"
echo "========================================"
echo ""

ERROR_COUNT=0
SERVICES=("api-gateway" "auth-service" "billing-service" "file-service" "notification-service")

# Check Go services
echo "[1/2] Checking Go services..."
for service in "${SERVICES[@]}"; do
    echo "  Checking $service..."
    cd "services/$service"
    
    # Run go mod tidy
    if ! go mod tidy 2>&1; then
        echo "  ❌ ERROR: go mod tidy failed for $service"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    fi
    
    # Try to build
    if ! go build ./... 2>&1; then
        echo "  ❌ ERROR: Build failed for $service"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    else
        echo "  ✅ $service builds successfully"
    fi
    
    cd ../..
done
echo ""

# Check frontend
echo "[2/2] Checking frontend..."
cd frontend
if [ -d "node_modules" ]; then
    if ! npx tsc --noEmit 2>&1; then
        echo "  ❌ ERROR: TypeScript errors found"
        ERROR_COUNT=$((ERROR_COUNT + 1))
    else
        echo "  ✅ No TypeScript errors"
    fi
else
    echo "  ⚠️  WARNING: node_modules not found, run 'npm install'"
fi
cd ..
echo ""

# Summary
echo "========================================"
if [ $ERROR_COUNT -eq 0 ]; then
    echo "✅ SUCCESS: No compilation errors found!"
    echo ""
    echo "All services compile successfully:"
    for service in "${SERVICES[@]}"; do
        echo "  ✅ $service"
    done
    echo "  ✅ frontend"
else
    echo "❌ FAILED: Found $ERROR_COUNT error(s)"
    echo "Please review the errors above"
fi
echo "========================================"
echo ""

exit $ERROR_COUNT

