#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="deploy/docker-compose.test.yml"
PROJECT_NAME="wessley-test"

cleanup() {
  echo "🧹 Tearing down test environment..."
  docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down -v --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "🚀 Starting test services..."
docker compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d --wait

echo "✅ All services healthy. Running integration tests..."
go test ./... -tags=integration -v -count=1 -timeout=120s

echo "🎉 Integration tests passed!"
