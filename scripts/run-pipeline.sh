#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(dirname "$0")"
DATA_DIR="${DATA_DIR:-/tmp/wessley-data}"
mkdir -p "$DATA_DIR"

# Build ingest
echo "Building ingest CLI..."
cd "$SCRIPT_DIR/.."
go build -o /tmp/ingest ./cmd/ingest

PIDS=()

cleanup() {
    echo "Stopping all processes..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    exit 0
}
trap cleanup SIGINT SIGTERM

# Start scrapers in background
echo "Starting scrapers..."
"$SCRIPT_DIR/run-scrapers.sh" &
PIDS+=($!)

# Give scrapers a head start
sleep 5

# Start ingest
echo "Starting ingestion pipeline..."
/tmp/ingest --dir "$DATA_DIR" --interval 30s &
PIDS+=($!)

echo ""
echo "=== Wessley MVP Pipeline Running ==="
echo "Scrapers: Reddit (5m), NHTSA/iFixit/Forums (30m)"
echo "Ingest: scanning $DATA_DIR every 30s"
echo "Data: Neo4j localhost:7687, Qdrant localhost:6334"
echo ""
echo "Press Ctrl+C to stop everything."

wait
