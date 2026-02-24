#!/usr/bin/env bash
set -euo pipefail

# Pull the Mistral model into the Ollama container on first run.
MODEL="${1:-mistral}"

echo "Waiting for Ollama to be ready..."
until curl -sf http://localhost:11434/api/tags >/dev/null 2>&1; do
  sleep 2
done

echo "Pulling model: $MODEL"
docker exec ollama ollama pull "$MODEL"
echo "Done — $MODEL is ready."
