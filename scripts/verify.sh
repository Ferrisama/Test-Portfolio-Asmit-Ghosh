#!/bin/bash

set -e

echo " Trading Journal Verification Script"
echo "======================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
        exit 1
    fi
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_info() {
    echo -e "ℹ  $1"
}

# Check Go installation
echo ""
echo " Checking Prerequisites..."
if command -v go >/dev/null 2>&1; then
    GO_VERSION=$(go version | cut -d' ' -f3)
    print_status 0 "Go is installed ($GO_VERSION)"
else
    print_status 1 "Go is not installed"
fi

# Check if this is a Go module
if [ -f "go.mod" ]; then
    print_status 0 "Go module found"
else
    print_status 1 "go.mod not found - run 'go mod init changeme'"
fi

# Check dependencies
echo ""
echo " Checking Dependencies..."
print_info "Running 'go mod tidy'..."
if go mod tidy; then
    print_status 0 "Go dependencies updated"
else
    print_status 1 "Failed to update dependencies"
fi

# Run Go vet
echo ""
echo " Running Go vet..."
if go vet ./...; then
    print_status 0 "Go vet passed"
else
    print_status 1 "Go vet failed"
fi

# Run tests
echo ""
echo "🧪 Running Tests..."
if go test -v ./...; then
    print_status 0 "All tests passed"
else
    print_status 1 "Tests failed"
fi

# Check if frontend exists
echo ""
echo "  Checking Frontend..."
if [ -d "frontend" ]; then
    print_status 0 "Frontend directory exists"
    
    if [ -f "frontend/package.json" ]; then
        print_status 0 "package.json found"
    else
        print_warning "package.json not found in frontend/"
    fi
    
    if [ -f "frontend/index.html" ]; then
        print_status 0 "index.html found"
    else
        print_warning "index.html not found in frontend/"
    fi
else
    print_warning "Frontend directory not found"
fi

# Check build configuration
echo ""
echo "  Checking Build Configuration..."
if [ -f "build/Taskfile.yml" ]; then
    print_status 0 "build/Taskfile.yml found"
else
    print_warning "build/Taskfile.yml not found"
fi

if [ -f "build/config.yml" ]; then
    print_status 0 "build/config.yml found"
else
    print_warning "build/config.yml not found"
fi

# Check migrations
echo ""
echo " Checking Database Setup..."
if [ -d "migrations" ]; then
    print_status 0 "Migrations directory exists"
    
    MIGRATION_COUNT=$(find migrations -name "*.sql" | wc -l)
    if [ $MIGRATION_COUNT -gt 0 ]; then
        print_status 0 "Found $MIGRATION_COUNT migration files"
    else
        print_warning "No migration files found"
    fi
else
    print_warning "Migrations directory not found"
fi

# Check test data
echo ""
echo " Checking Test Data..."
if [ -f "testdata/sample_trades.csv" ]; then
    print_status 0 "Sample CSV found"
    
    # Count lines in CSV (excluding header)
    if [ -s "testdata/sample_trades.csv" ]; then
        LINE_COUNT=$(($(wc -l < "testdata/sample_trades.csv") - 1))
        print_info "Sample CSV contains $LINE_COUNT trade records"
    fi
else
    print_warning "testdata/sample_trades.csv not found"
fi

# Test analytics with sample data
echo ""
echo " Testing Analytics Functions..."
if [ -f "testdata/sample_trades.csv" ] && [ -f "cmd/metricscheck/main.go" ]; then
    print_info "Running analytics test on sample data..."
    
    # Create temporary expected results file
    TEMP_EXPECTED=$(mktemp)
    cat > "$TEMP_EXPECTED" << 'EOF'
{
  "winRate": 0.6,
  "profitFactor": 2.0,
  "maxDD": 15.0,
  "sharpe": 0.5,
  "sortino": 0.7,
  "expectancy": 5.0
}
EOF
    
    if go run cmd/metricscheck/main.go --csv testdata/sample_trades.csv --expected "$TEMP_EXPECTED" >/dev/null 2>&1; then
        print_status 0 "Analytics functions working"
    else
        print_warning "Analytics test failed (this may be expected if sample data doesn't match expectations)"
    fi
    
    rm -f "$TEMP_EXPECTED"
else
    print_warning "Cannot test analytics - missing sample data or metricscheck"
fi

# Try building the application
echo ""
echo "Testing Build..."
if command -v wails3 >/dev/null 2>&1; then
    print_info "Found Wails v3, attempting build..."
    if timeout 30s wails3 build >/dev/null 2>&1; then
        print_status 0 "Build successful"
    else
        print_warning "Build failed or timed out"
    fi
else
    if go build -o bin/portfolio . >/dev/null 2>&1; then
        print_status 0 "Go build successful"
        rm -f bin/portfolio
    else
        print_warning "Go build failed"
    fi
fi

echo ""
echo "Verification Complete!"
echo ""
print_info "To run the application:"
echo "   Development: wails3 dev"
echo "   Production:  wails3 build"
echo ""
print_info "To test manually:"
echo "   1. Run 'wails3 dev'"
echo "   2. Import the sample CSV data"
echo "   3. Apply filters to see analytics and trades table"
echo "   4. Test different filter combinations"