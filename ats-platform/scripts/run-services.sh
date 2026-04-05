#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

SERVICES=(
  "resume-service:./cmd/resume-service:resume-service"
  "interview-service:./cmd/interview-service:interview-service"
  "search-service:./cmd/search-service:search-service"
)

PIDS=()
START_INFRA=1
BUILD_ONLY=0
GOCACHE_DIR="${ROOT_DIR}/.gocache"
LOCAL_CONSUL_DOCKER=1

usage() {
  cat <<'EOF'
Usage: scripts/run-services.sh [options]

Build and run resume-service, interview-service, and search-service together.
Optionally starts docker-compose infrastructure first.

Options:
  --no-infra     Do not start docker-compose dependencies
  --build-only   Only build the three binaries, do not run them
  --gateway      Also build and run the API gateway
  --host-consul  Assume Consul is running on the host instead of Docker and do not default SERVICE_ADDRESS
  -h, --help     Show this help
EOF
}

log() {
  printf '[run-services] %s\n' "$*"
}

require_port_free() {
  local port="$1"
  if nc -z 127.0.0.1 "$port" >/dev/null 2>&1; then
    echo "Port $port is already in use. Stop the existing process before running run-services.sh." >&2
    exit 1
  fi
}

check_required_ports() {
  local ports=(8081 9090 8082 9091 8083)

  if printf '%s\n' "${SERVICES[@]}" | grep -q '^gateway:'; then
    ports+=(8080)
  fi

  for port in "${ports[@]}"; do
    require_port_free "$port"
  done
}

cleanup() {
  local exit_code=$?

  trap - SIGINT SIGTERM EXIT

  if ((${#PIDS[@]} > 0)); then
    log "stopping service processes..."
    for pid in "${PIDS[@]}"; do
      if kill -0 "$pid" 2>/dev/null; then
        kill "$pid" 2>/dev/null || true
      fi
    done

    for pid in "${PIDS[@]}"; do
      wait "$pid" 2>/dev/null || true
    done
  fi

  log "all managed services stopped"
  exit "$exit_code"
}

while (($# > 0)); do
  case "$1" in
    --no-infra)
      START_INFRA=0
      ;;
    --build-only)
      BUILD_ONLY=1
      ;;
    --gateway)
      SERVICES+=("gateway:./cmd/gateway:gateway")
      ;;
    --host-consul)
      LOCAL_CONSUL_DOCKER=0
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
  shift
done

mkdir -p "$GOCACHE_DIR"

export CONSUL_HOST="${CONSUL_HOST:-127.0.0.1}"
export CONSUL_PORT="${CONSUL_PORT:-8500}"
export DB_HOST="${DB_HOST:-127.0.0.1}"
export DB_PORT="${DB_PORT:-5432}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
export MINIO_ENDPOINT="${MINIO_ENDPOINT:-127.0.0.1:9000}"
export ES_ADDRESSES="${ES_ADDRESSES:-http://127.0.0.1:9200}"

if [[ -z "${SERVICE_ADDRESS:-}" ]] && [[ "${CONSUL_HOST}" == "127.0.0.1" ]]; then
  export SERVICE_ADDRESS="host.docker.internal"
  log "defaulting SERVICE_ADDRESS=host.docker.internal for host-run services with Docker Consul"
fi

if ((START_INFRA)); then
  log "starting docker-compose dependencies..."
  docker-compose -f deployments/docker-compose.yml up -d
fi

log "building service binaries..."
for service in "${SERVICES[@]}"; do
  IFS=':' read -r name cmd_path output_name <<<"$service"
  log "building ${name}..."
  GOCACHE="$GOCACHE_DIR" go build -o "$output_name" "$cmd_path"
done

if ((BUILD_ONLY)); then
  log "build completed"
  exit 0
fi

check_required_ports

trap cleanup SIGINT SIGTERM EXIT

if ((LOCAL_CONSUL_DOCKER)) && [[ -z "${SERVICE_ADDRESS:-}" ]]; then
  export SERVICE_ADDRESS="host.docker.internal"
  log "defaulting SERVICE_ADDRESS=${SERVICE_ADDRESS} for host-run services with Docker Consul"
fi

export CONSUL_HOST="${CONSUL_HOST:-127.0.0.1}"
export CONSUL_PORT="${CONSUL_PORT:-8500}"
export DB_HOST="${DB_HOST:-127.0.0.1}"
export DB_PORT="${DB_PORT:-5432}"
export DB_USER="${DB_USER:-postgres}"
export DB_PASSWORD="${DB_PASSWORD:-postgres}"
export DB_NAME="${DB_NAME:-ats}"
export REDIS_ADDR="${REDIS_ADDR:-127.0.0.1:6379}"
export MINIO_ENDPOINT="${MINIO_ENDPOINT:-127.0.0.1:9000}"
export MINIO_USER="${MINIO_USER:-minioadmin}"
export MINIO_PASSWORD="${MINIO_PASSWORD:-minioadmin}"
export MINIO_BUCKET="${MINIO_BUCKET:-resumes}"
export MINIO_USE_SSL="${MINIO_USE_SSL:-false}"
export ES_ADDRESSES="${ES_ADDRESSES:-http://127.0.0.1:9200}"

for service in "${SERVICES[@]}"; do
  IFS=':' read -r name _ output_name <<<"$service"
  log "starting ${name}..."
  "./${output_name}" &
  PIDS+=("$!")
done

log "services are running; press Ctrl+C to stop all"

while true; do
  for pid in "${PIDS[@]}"; do
    if ! kill -0 "$pid" 2>/dev/null; then
      log "a managed service exited unexpectedly"
      exit 1
    fi
  done
  sleep 1
done
