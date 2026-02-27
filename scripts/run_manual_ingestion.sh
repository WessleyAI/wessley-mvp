#!/bin/bash
# Process all PDFs in a manual directory through the intelligent ingestion pipeline.
# Usage: ./scripts/run_manual_ingestion.sh [manual_dir] [output_dir]

set -euo pipefail

MANUAL_DIR="${1:-/tmp/wessley-data/manuals}"
OUTPUT_DIR="${2:-/tmp/wessley-data}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== Wessley Manual Ingestion Pipeline ==="
echo "Manual dir: $MANUAL_DIR"
echo "Output dir: $OUTPUT_DIR"
echo ""

mkdir -p "$OUTPUT_DIR"

count=0
find "$MANUAL_DIR" -name "*.pdf" -type f | while read -r pdf; do
    echo "Processing: $pdf"
    python3 "$SCRIPT_DIR/manual_worker.py" "$pdf" "$OUTPUT_DIR" || {
        echo "  WARNING: Failed to process $pdf"
        continue
    }
    count=$((count + 1))
done

echo ""
echo "=== Done. Output files: ==="
ls "$OUTPUT_DIR"/manual-*.json 2>/dev/null | wc -l | xargs echo "Total JSON files:"
