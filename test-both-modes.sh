#!/bin/bash
# Test script to verify both standalone service and embedded package work correctly

set -e

echo "======================================"
echo "Rebound - Testing Both Deployment Modes"
echo "======================================"
echo

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Test 1: Build standalone service
echo "Test 1: Building standalone HTTP service..."
if go build -o /tmp/rebound-standalone ./cmd/rebound; then
    echo -e "${GREEN}✓${NC} Standalone service builds successfully"
else
    echo -e "${RED}✗${NC} Standalone service build failed"
    exit 1
fi
echo

# Test 2: Build embedded package
echo "Test 2: Building embedded package..."
if go build ./pkg/rebound; then
    echo -e "${GREEN}✓${NC} Embedded package builds successfully"
else
    echo -e "${RED}✗${NC} Embedded package build failed"
    exit 1
fi
echo

# Test 3: Run all tests
echo "Test 3: Running all unit tests..."
if go test ./internal/... ./pkg/... -cover > /tmp/test-output.txt 2>&1; then
    echo -e "${GREEN}✓${NC} All tests pass"
    echo
    echo "Coverage summary:"
    grep "coverage:" /tmp/test-output.txt | grep -v "0.0%"
else
    echo -e "${RED}✗${NC} Some tests failed"
    cat /tmp/test-output.txt
    exit 1
fi
echo

# Test 4: Verify no kafkaretry references
echo "Test 4: Checking for old 'kafkaretry' references..."
if grep -r "kafkaretry/" --include="*.go" . 2>/dev/null | grep -v "Binary file"; then
    echo -e "${RED}✗${NC} Found kafkaretry references in Go files"
    exit 1
else
    echo -e "${GREEN}✓${NC} No kafkaretry references found in Go files"
fi
echo

# Test 5: Check module name
echo "Test 5: Verifying module name..."
if grep "^module rebound$" go.mod > /dev/null; then
    echo -e "${GREEN}✓${NC} Module name is correct: rebound"
else
    echo -e "${RED}✗${NC} Module name is incorrect"
    exit 1
fi
echo

# Test 6: Verify package structure
echo "Test 6: Verifying package structure..."
REQUIRED_DIRS=(
    "pkg/rebound"
    "internal/domain"
    "internal/adapter/primary/http"
    "internal/adapter/primary/worker"
    "internal/adapter/secondary/kafkaproducer"
    "internal/adapter/secondary/httpproducer"
    "internal/adapter/secondary/producerfactory"
    "internal/adapter/secondary/redisstore"
    "cmd/rebound"
)

for dir in "${REQUIRED_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        echo -e "${GREEN}✓${NC} $dir exists"
    else
        echo -e "${RED}✗${NC} $dir missing"
        exit 1
    fi
done
echo

# Test 7: Verify documentation
echo "Test 7: Verifying documentation files..."
REQUIRED_DOCS=(
    "README.md"
    "QUICKSTART.md"
    "DEPLOYMENT.md"
    "INTEGRATION.md"
    "pkg/rebound/README.md"
    "examples/EXAMPLES.md"
)

for doc in "${REQUIRED_DOCS[@]}"; do
    if [ -f "$doc" ]; then
        echo -e "${GREEN}✓${NC} $doc exists"
    else
        echo -e "${RED}✗${NC} $doc missing"
        exit 1
    fi
done
echo

# Test 8: Check example files
echo "Test 8: Checking example files..."
EXAMPLE_DIRS=(
    "examples/01-basic-usage"
    "examples/02-email-service"
    "examples/03-webhook-delivery"
    "examples/04-di-integration"
    "examples/05-payment-processing"
    "examples/06-multi-tenant"
)

for dir in "${EXAMPLE_DIRS[@]}"; do
    if [ -d "$dir" ] && [ -f "$dir/main.go" ]; then
        echo -e "${GREEN}✓${NC} $dir exists with main.go"
    else
        echo -e "${RED}✗${NC} $dir missing or incomplete"
        exit 1
    fi
done
echo

# Summary
echo "======================================"
echo -e "${GREEN}All Tests Passed!${NC}"
echo "======================================"
echo
echo "Both deployment modes are working:"
echo "  ✓ Standalone HTTP Service (cmd/rebound)"
echo "  ✓ Embedded Go Package (pkg/rebound)"
echo
echo "Next steps:"
echo "  • Run standalone: go run cmd/rebound/main.go"
echo "  • Try examples: go run examples/01-basic-usage/main.go"
echo "  • Read docs: QUICKSTART.md, DEPLOYMENT.md"
echo

# Cleanup
rm -f /tmp/rebound-standalone /tmp/test-output.txt
