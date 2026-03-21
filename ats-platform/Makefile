.PHONY: infra-up infra-down proto migrate-up test run-gateway run-resume run-interview run-search run-all clean

# Infrastructure
infra-up:
	docker-compose -f deployments/docker-compose.yml up -d

infra-down:
	docker-compose -f deployments/docker-compose.yml down

infra-logs:
	docker-compose -f deployments/docker-compose.yml logs -f

# Database
migrate-up:
	@echo "Run migrations manually or use migrate tool"

# Protobuf
proto:
	protoc --go_out=. --go-grpc_out=. proto/*.proto

# Services
run-gateway:
	go run ./cmd/gateway

run-resume:
	go run ./cmd/resume-service

run-interview:
	go run ./cmd/interview-service

run-search:
	go run ./cmd/search-service

run-all:
	@echo "Starting all services..."
	@go run ./cmd/resume-service &
	@go run ./cmd/interview-service &
	@go run ./cmd/search-service &
	@go run ./cmd/gateway

# Development
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	go clean