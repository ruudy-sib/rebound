#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Rebound Comprehensive Benchmark Suite ===${NC}\n"

# Check if services are running
echo -e "${YELLOW}Checking services...${NC}"
if ! redis-cli -h localhost -p 6379 PING > /dev/null 2>&1; then
    echo -e "${YELLOW}Redis not running. Starting services...${NC}"
    cd ../.. && docker-compose up -d redis kafka
    sleep 5
fi
echo -e "${GREEN}✓ Services are running${NC}\n"

# Create results directory
RESULTS_DIR="benchmark-results-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}Results will be saved to: ${RESULTS_DIR}${NC}\n"

# Function to run benchmark and save results
run_benchmark() {
    local name=$1
    local bench=$2
    local file="${RESULTS_DIR}/${name}.txt"

    echo -e "${YELLOW}Running ${name}...${NC}"
    cd ../../pkg/rebound
    go test -bench="$bench" -benchmem -benchtime=10s > "$file" 2>&1
    cd - > /dev/null
    echo -e "${GREEN}✓ ${name} complete${NC}\n"
}

# Run benchmarks
echo -e "${BLUE}1. Task Creation Benchmarks${NC}"
run_benchmark "01-task-creation-kafka" "BenchmarkTaskCreation$"
run_benchmark "02-task-creation-http" "BenchmarkTaskCreationHTTP"

echo -e "${BLUE}2. Parallel Performance${NC}"
run_benchmark "03-parallel-creation" "BenchmarkTaskCreationParallel"

echo -e "${BLUE}3. Variable Payload Sizes${NC}"
run_benchmark "04-payload-sizes" "BenchmarkTaskCreationWithVariablePayload"

echo -e "${BLUE}4. Retry Configurations${NC}"
run_benchmark "05-retry-configs" "BenchmarkTaskCreationWithRetries"

echo -e "${BLUE}5. Initialization${NC}"
run_benchmark "06-initialization" "BenchmarkReboundInitialization"

# Generate summary report
echo -e "${BLUE}Generating summary report...${NC}"

cat > "${RESULTS_DIR}/summary.md" << 'EOF'
# Rebound Benchmark Results

Generated: $(date +"%Y-%m-%d %H:%M:%S")

## System Information

- Go Version: $(go version)
- OS: $(uname -s)
- Architecture: $(uname -m)
- CPU Cores: $(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo "unknown")

## Benchmark Results

EOF

# Add results to summary
for file in ${RESULTS_DIR}/*.txt; do
    if [ -f "$file" ]; then
        echo -e "\n### $(basename "$file" .txt)\n" >> "${RESULTS_DIR}/summary.md"
        echo '```' >> "${RESULTS_DIR}/summary.md"
        grep "^Benchmark" "$file" >> "${RESULTS_DIR}/summary.md" || true
        echo '```' >> "${RESULTS_DIR}/summary.md"
    fi
done

# Add performance summary
cat >> "${RESULTS_DIR}/summary.md" << 'EOF'

## Performance Summary

### Key Metrics

- **Task Creation Rate:** See benchmark results above
- **Memory Usage:** Check ns/op and B/op columns
- **Allocations:** Check allocs/op column

### Recommendations

1. **Throughput Optimization:**
   - Use parallel task creation for high-volume scenarios
   - Keep payload sizes under 10KB when possible
   - Use appropriate batch sizes for bulk operations

2. **Memory Optimization:**
   - Monitor allocations per operation
   - Consider connection pooling for high-frequency operations
   - Review payload sizes if memory usage is high

3. **Retry Configuration:**
   - Use fewer retries (1-3) for non-critical operations
   - Use more retries (5-10) for critical operations
   - Balance retry count with total system load

## Next Steps

1. Review individual benchmark files in this directory
2. Compare results with your production requirements
3. Adjust configurations based on findings
4. Re-run benchmarks after optimizations

EOF

echo -e "${GREEN}✓ Summary report generated${NC}\n"

# Display summary
echo -e "${BLUE}=== Benchmark Summary ===${NC}\n"
cat "${RESULTS_DIR}/summary.md"

echo -e "\n${GREEN}✓ All benchmarks complete!${NC}"
echo -e "${BLUE}Results saved to: ${RESULTS_DIR}/${NC}\n"

# Create a latest symlink
rm -f benchmark-results-latest
ln -s "$RESULTS_DIR" benchmark-results-latest
echo -e "${YELLOW}Tip: View latest results with: cat benchmark-results-latest/summary.md${NC}"
