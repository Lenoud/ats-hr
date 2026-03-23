.PHONY: infra-up infra-down proto migrate-up test run-gateway run-resume run-interview run-search run-all run-all-no-infra run-all-with-gateway build-services clean

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
	protoc --go_out=. --go_opt=module=github.com/example/ats-platform --go-grpc_out=. --go-grpc_opt=module=github.com/example/ats-platform proto/*.proto

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
	./scripts/run-services.sh

run-all-no-infra:
	./scripts/run-services.sh --no-infra

run-all-with-gateway:
	./scripts/run-services.sh --gateway

build-services:
	./scripts/run-services.sh --build-only

# Development
test:
	go test ./... -v

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

clean:
	rm -rf bin/
	go clean
