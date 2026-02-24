APP_NAME := parallelis
GO_SRC := ./src/main/go/com/demo/jobprocessor
DOCKER_IMAGE := $(APP_NAME):latest

.PHONY: all build run test bench profile lint clean docker deploy

all: lint test build

# Build the application binary
build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/$(APP_NAME) $(GO_SRC)/main.go

# Run the application locally
run:
	go run $(GO_SRC)/main.go

# Run all tests
test:
	go test ./... -v -count=1

# Run benchmarks with memory allocation stats
bench:
	@echo "Running benchmarks..."
	go test $(GO_SRC)/service -bench=. -benchmem -count=3

# Generate CPU and memory profiles for analysis
# Usage: make profile && go tool pprof cpu.prof
profile:
	@echo "Generating CPU and memory profiles..."
	go test $(GO_SRC)/service -bench=. -benchmem \
		-cpuprofile=profiles/cpu.prof \
		-memprofile=profiles/mem.prof
	@mkdir -p profiles
	@echo "Profiles saved to profiles/"
	@echo "Analyze: go tool pprof profiles/cpu.prof"

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -rf bin/ profiles/
	go clean

# Build Docker image (multi-stage)
docker:
	docker build -t $(DOCKER_IMAGE) -f Dockerfile .

# Deploy to AWS ECS via Terraform
deploy:
	cd terraform && terraform init && terraform apply

# Destroy AWS infrastructure
destroy:
	cd terraform && terraform destroy

# Run load test (requires locust)
loadtest:
	locust -f experiment_partitions.py --host=http://localhost:8080

# Show help
help:
	@echo "Available targets:"
	@echo "  build    - Build the application"
	@echo "  run      - Run locally"
	@echo "  test     - Run tests"
	@echo "  bench    - Run benchmarks"
	@echo "  profile  - Generate CPU/memory profiles"
	@echo "  lint     - Run linter"
	@echo "  docker   - Build Docker image"
	@echo "  deploy   - Deploy to AWS"
	@echo "  loadtest - Run Locust load test"