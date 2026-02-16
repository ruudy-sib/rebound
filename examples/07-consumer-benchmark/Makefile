.PHONY: help run benchmark clean docker-up docker-down

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

docker-up: ## Start required services (Redis, Kafka)
	@echo "Starting Redis and Kafka..."
	cd ../.. && docker-compose up -d redis kafka
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "✓ Services ready"

docker-down: ## Stop all services
	@echo "Stopping services..."
	cd ../.. && docker-compose down
	@echo "✓ Services stopped"

run: ## Run consumer benchmark with default settings
	@echo "Running consumer benchmark..."
	go run .

run-low-failure: ## Run with 10% failure rate
	@echo "Running with low failure rate (10%)..."
	go run . -failure-rate 0.1 -duration 2m

run-high-failure: ## Run with 50% failure rate
	@echo "Running with high failure rate (50%)..."
	go run . -failure-rate 0.5 -duration 2m

run-long: ## Run extended benchmark (5 minutes)
	@echo "Running extended benchmark..."
	go run . -duration 5m

benchmark: ## Run Go benchmarks
	@echo "Running benchmarks..."
	cd ../../pkg/rebound && go test -bench=. -benchmem -benchtime=5s

benchmark-parallel: ## Run parallel benchmarks only
	@echo "Running parallel benchmarks..."
	cd ../../pkg/rebound && go test -bench=Parallel -benchmem -benchtime=10s

benchmark-payload: ## Run payload size benchmarks
	@echo "Running payload size benchmarks..."
	cd ../../pkg/rebound && go test -bench=Payload -benchmem

benchmark-all: ## Run comprehensive benchmark suite
	@echo "Running comprehensive benchmark suite..."
	@./run-benchmarks.sh

clean: ## Clean up generated files
	@echo "Cleaning up..."
	@rm -f *.out *.test
	@echo "✓ Clean complete"

test: ## Run tests
	@echo "Running tests..."
	go test -v -race ./...

deps: ## Install dependencies
	@echo "Installing dependencies..."
	go mod download
	@echo "✓ Dependencies installed"
