#!/bin/bash
set -e
cd "$(dirname "$0")/.."
docker compose -f docker-compose.observability.yml up -d
echo "Grafana:    http://localhost:3001 (admin/wessley)"
echo "Prometheus: http://localhost:9090"
