#!/usr/bin/env bash
set -e

DATA_DIR="${DATA_DIR:-/tmp/wessley-data}"
mkdir -p "$DATA_DIR"

DATE=$(date +%Y-%m-%d)
REDDIT_OUT="$DATA_DIR/reddit-$DATE.json"
SOURCES_OUT="$DATA_DIR/sources-$DATE.json"
YOUTUBE_OUT="$DATA_DIR/youtube-$DATE.json"

# Build scrapers
echo "Building scrapers..."
cd "$(dirname "$0")/.."
go build -o /tmp/scraper-reddit ./cmd/scraper-reddit
go build -o /tmp/scraper-sources ./cmd/scraper-sources
go build -o /tmp/scraper-youtube ./cmd/scraper-youtube

PIDS=()

cleanup() {
    echo "Stopping scrapers..."
    for pid in "${PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
    done
    exit 0
}
trap cleanup SIGINT SIGTERM

# Reddit scraper: 5min interval, more subreddits, higher limit
echo "Starting Reddit scraper (5m interval) -> $REDDIT_OUT"
/tmp/scraper-reddit --interval 5m --limit 50 >> "$REDDIT_OUT" 2>>"$DATA_DIR/scraper.log" &
PIDS+=($!)

# Sources scraper: NHTSA + iFixit + forums, more makes, 30min interval
echo "Starting sources scraper (30m interval) -> $SOURCES_OUT"
/tmp/scraper-sources --interval 30m \
    --nhtsa-makes "TOYOTA,HONDA,FORD,CHEVROLET,BMW,NISSAN,HYUNDAI,KIA,SUBARU,MAZDA,VOLKSWAGEN,MERCEDES-BENZ,AUDI,JEEP,RAM,GMC,DODGE" \
    --nhtsa-year 2024 \
    >> "$SOURCES_OUT" 2>>"$DATA_DIR/scraper.log" &
PIDS+=($!)

# YouTube scraper: popular automotive repair channels, runs every 60min
# These are well-known automotive repair video IDs (ChrisFix, Scotty Kilmer, etc.)
echo "Starting YouTube scraper (60m interval) -> $YOUTUBE_OUT"
(
    while true; do
        # Scrape popular automotive repair videos by ID (no API key needed)
        # ChrisFix, Scotty Kilmer, South Main Auto, Pine Hollow Auto Diagnostics, etc.
        /tmp/scraper-youtube --video-ids \
            "IXCzl0Mj2gU,5nQnujWGr_8,kVK7YWFEMIQ,_CLz4MpmMiY,O1hF25Cowv8,\
j5v8D-alAKE,drbhNLvYxGQ,ENWlBp97PCA,AtG6MRyYjEo,BQSCsaGQ1GY,\
3_3Hp9VqSSI,Bqm3u4hFvzI,vC8LbvYk6es,mkMcJWnEfZ0,axJm-F_CZk8,\
0fkhVjDR09s,rHBDDo7mfLI,HU43EerSnaw,fLQJEhVWvyE,n4vusY2BWAM" \
            >> "$YOUTUBE_OUT" 2>>"$DATA_DIR/scraper.log"
        echo "YouTube batch done, sleeping 60m..." >&2
        sleep 3600
    done
) &
PIDS+=($!)

echo "Scrapers running. PIDs: ${PIDS[*]}"
echo "  Reddit:  5m interval, 50 posts/sub"
echo "  Sources: 30m interval, 17 makes"
echo "  YouTube: 60m interval, video IDs"
echo "Press Ctrl+C to stop."

# Wait forever
wait
