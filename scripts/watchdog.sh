#!/usr/bin/env bash
# Wessley Process Watchdog
# Monitors ingest + scraper processes, restarts on death, logs events.
# Usage: ./scripts/watchdog.sh [--interval 60]
#
# Designed to run in background: nohup ./scripts/watchdog.sh &

set -u

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DATA_DIR="${DATA_DIR:-/tmp/wessley-data}"
LOG="$DATA_DIR/watchdog.log"
PIDFILE_INGEST="$DATA_DIR/.ingest.pid"
PIDFILE_SCRAPER="$DATA_DIR/.scraper.pid"
CHECK_INTERVAL="${1:-60}"  # seconds between checks
MAX_RESTARTS=5             # max restarts per process per hour
RESTART_WINDOW=3600        # 1 hour window for restart counting

mkdir -p "$DATA_DIR"

# Track restart timestamps
declare -a INGEST_RESTARTS=()
declare -a SCRAPER_RESTARTS=()

log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*" | tee -a "$LOG"
}

is_alive() {
    local pid="$1"
    [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null
}

read_pid() {
    local pidfile="$1"
    if [ -f "$pidfile" ]; then
        cat "$pidfile"
    fi
}

save_pid() {
    echo "$2" > "$1"
}

count_recent_restarts() {
    local -n arr=$1
    local now cutoff count=0
    now=$(date +%s)
    cutoff=$((now - RESTART_WINDOW))
    local new_arr=()
    for ts in "${arr[@]+"${arr[@]}"}"; do
        if [ "$ts" -ge "$cutoff" ]; then
            count=$((count + 1))
            new_arr+=("$ts")
        fi
    done
    eval "$1=(\"\${new_arr[@]+\"\${new_arr[@]}\"}\")"
    echo "$count"
}

build_binaries() {
    log "Building binaries..."
    cd "$REPO_DIR"
    go build -o /tmp/ingest ./cmd/ingest 2>>"$LOG" && \
    go build -o /tmp/scraper-sources ./cmd/scraper-sources 2>>"$LOG"
    return $?
}

start_ingest() {
    local restarts
    restarts=$(count_recent_restarts INGEST_RESTARTS)
    if [ "$restarts" -ge "$MAX_RESTARTS" ]; then
        log "⛔ INGEST: $restarts restarts in last hour — backing off"
        return 1
    fi

    log "🚀 Starting ingest..."
    /tmp/ingest --dir "$DATA_DIR" --interval 30s >> "$DATA_DIR/ingest.log" 2>&1 &
    local pid=$!
    save_pid "$PIDFILE_INGEST" "$pid"
    INGEST_RESTARTS+=("$(date +%s)")
    log "✅ Ingest started (PID $pid)"
}

start_scraper() {
    local restarts
    restarts=$(count_recent_restarts SCRAPER_RESTARTS)
    if [ "$restarts" -ge "$MAX_RESTARTS" ]; then
        log "⛔ SCRAPER: $restarts restarts in last hour — backing off"
        return 1
    fi

    local DATE
    DATE=$(date +%Y-%m-%d)
    log "🚀 Starting scraper..."
    /tmp/scraper-sources --interval 30m \
        --nhtsa-makes "TOYOTA,HONDA,FORD,CHEVROLET,BMW,NISSAN,HYUNDAI,KIA,SUBARU,MAZDA,VOLKSWAGEN,MERCEDES-BENZ,AUDI,JEEP,RAM,GMC,DODGE" \
        --nhtsa-year 2020 \
        >> "$DATA_DIR/sources-$DATE.json" 2>>"$DATA_DIR/scraper.log" &
    local pid=$!
    save_pid "$PIDFILE_SCRAPER" "$pid"
    SCRAPER_RESTARTS+=("$(date +%s)")
    log "✅ Scraper started (PID $pid)"
}

cleanup() {
    log "🛑 Watchdog shutting down"
    exit 0
}
trap cleanup SIGINT SIGTERM

# --- Main loop ---

log "🐕 Watchdog started (check every ${CHECK_INTERVAL}s, max ${MAX_RESTARTS} restarts/hr)"

# Build once on start
if [ ! -f /tmp/ingest ] || [ ! -f /tmp/scraper-sources ]; then
    build_binaries || { log "❌ Build failed, exiting"; exit 1; }
fi

# Pick up existing PIDs if processes are already running
INGEST_PID=$(read_pid "$PIDFILE_INGEST")
SCRAPER_PID=$(read_pid "$PIDFILE_SCRAPER")

if is_alive "$INGEST_PID"; then
    log "📎 Ingest already running (PID $INGEST_PID)"
else
    start_ingest
    INGEST_PID=$(read_pid "$PIDFILE_INGEST")
fi

if is_alive "$SCRAPER_PID"; then
    log "📎 Scraper already running (PID $SCRAPER_PID)"
else
    start_scraper
    SCRAPER_PID=$(read_pid "$PIDFILE_SCRAPER")
fi

while true; do
    sleep "$CHECK_INTERVAL"

    # Check ingest
    INGEST_PID=$(read_pid "$PIDFILE_INGEST")
    if ! is_alive "$INGEST_PID"; then
        log "💀 Ingest died (was PID $INGEST_PID)"
        start_ingest
        INGEST_PID=$(read_pid "$PIDFILE_INGEST")
    fi

    # Check scraper
    SCRAPER_PID=$(read_pid "$PIDFILE_SCRAPER")
    if ! is_alive "$SCRAPER_PID"; then
        log "💀 Scraper died (was PID $SCRAPER_PID)"
        start_scraper
        SCRAPER_PID=$(read_pid "$PIDFILE_SCRAPER")
    fi
done
