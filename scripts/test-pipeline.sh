#!/usr/bin/env bash
# Integration test: drop a sample NHTSA JSON file, verify ingest picks it up.
set -euo pipefail

DATA_DIR="${DATA_DIR:-/tmp/wessley-data}"
NEO4J_URL="${NEO4J_URL:-bolt://localhost:7687}"
NEO4J_USER="${NEO4J_USER:-neo4j}"
NEO4J_PASS="${NEO4J_PASS:-wessley123}"
QDRANT_URL="${QDRANT_URL:-http://localhost:6333}"
COLLECTION="${COLLECTION:-wessley}"
TIMEOUT="${TIMEOUT:-60}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; FAILURES=$((FAILURES+1)); }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

FAILURES=0
TEST_ID="test-$(date +%s)"

info "Test ID: $TEST_ID"
info "Data dir: $DATA_DIR"

# Step 1: Create sample NHTSA JSON file
mkdir -p "$DATA_DIR"
cat > "$DATA_DIR/test-pipeline-${TEST_ID}.json" <<EOF
{
  "source": "nhtsa",
  "source_id": "${TEST_ID}",
  "title": "2019 Toyota Camry Engine Stalling",
  "content": "The 2019 Toyota Camry with 2.5L engine stalls at low speeds. The engine control module may need reprogramming. TSB issued for ECU firmware update.",
  "author": "NHTSA",
  "url": "https://www.nhtsa.gov/complaints/${TEST_ID}",
  "published_at": "2024-01-15T00:00:00Z",
  "scraped_at": "2026-02-27T12:00:00Z",
  "metadata": {
    "vehicle": "2019 Toyota Camry",
    "vehicle_info": {"make": "Toyota", "model": "Camry", "year": 2019},
    "components": "ENGINE, ECU",
    "symptoms": ["stalling", "engine dies"],
    "keywords": ["engine", "stall", "ecu", "firmware"]
  }
}
EOF
pass "Created sample JSON file: test-pipeline-${TEST_ID}.json"

# Step 2: Wait for ingest to pick it up
info "Waiting up to ${TIMEOUT}s for ingest to process the file..."
DOC_ID="nhtsa:${TEST_ID}"
ELAPSED=0
FOUND=false

while [ $ELAPSED -lt $TIMEOUT ]; do
  # Check if the file was processed by looking at the ingest state
  if [ -f "$DATA_DIR/.ingest-state.json" ]; then
    if grep -q "test-pipeline-${TEST_ID}" "$DATA_DIR/.ingest-state.json" 2>/dev/null; then
      FOUND=true
      break
    fi
  fi
  sleep 2
  ELAPSED=$((ELAPSED+2))
done

if [ "$FOUND" = true ]; then
  pass "Ingest processed the file (${ELAPSED}s)"
else
  fail "Ingest did not process file within ${TIMEOUT}s (is cmd/ingest running?)"
  info "Continuing with direct checks anyway..."
fi

# Step 3: Query Neo4j for nodes + relationships
info "Querying Neo4j for test document..."
NEO4J_RESULT=$(curl -s -u "${NEO4J_USER}:${NEO4J_PASS}" \
  -H "Content-Type: application/json" \
  -d "{\"statements\": [{\"statement\": \"MATCH (c:Component {id: '${DOC_ID}'}) RETURN c.id AS id, c.name AS name\"}]}" \
  "${NEO4J_URL/bolt/http}:7474/db/neo4j/tx/commit" 2>/dev/null || echo '{"results":[{"data":[]}]}')

if echo "$NEO4J_RESULT" | grep -q "$DOC_ID"; then
  pass "Neo4j: Found Component node for $DOC_ID"
else
  fail "Neo4j: Component node not found for $DOC_ID"
fi

# Check for vehicle hierarchy relationships
info "Querying Neo4j for vehicle hierarchy relationships..."
REL_RESULT=$(curl -s -u "${NEO4J_USER}:${NEO4J_PASS}" \
  -H "Content-Type: application/json" \
  -d '{"statements": [{"statement": "MATCH (mk:Make {id: \"toyota\"})-[:HAS_MODEL]->(m:VehicleModel)-[:OF_MODEL]-(my:ModelYear) WHERE my.year = 2019 RETURN mk.name AS make, m.name AS model, my.year AS year"}]}' \
  "${NEO4J_URL/bolt/http}:7474/db/neo4j/tx/commit" 2>/dev/null || echo '{"results":[{"data":[]}]}')

if echo "$REL_RESULT" | grep -q "Toyota"; then
  pass "Neo4j: Found Make→Model→ModelYear hierarchy for Toyota Camry 2019"
else
  fail "Neo4j: Vehicle hierarchy not found"
fi

# Check for any relationships
REL_COUNT=$(curl -s -u "${NEO4J_USER}:${NEO4J_PASS}" \
  -H "Content-Type: application/json" \
  -d '{"statements": [{"statement": "MATCH ()-[r]->() RETURN count(r) AS cnt"}]}' \
  "${NEO4J_URL/bolt/http}:7474/db/neo4j/tx/commit" 2>/dev/null || echo '{"results":[{"data":[]}]}')

info "Neo4j relationship count: $(echo "$REL_COUNT" | grep -o '"cnt":[0-9]*' | head -1 || echo 'unknown')"

# Step 4: Query Qdrant for the vector
info "Querying Qdrant for test vectors..."
QDRANT_RESULT=$(curl -s -X POST "${QDRANT_URL}/collections/${COLLECTION}/points/scroll" \
  -H "Content-Type: application/json" \
  -d "{\"filter\": {\"must\": [{\"key\": \"doc_id\", \"match\": {\"value\": \"${DOC_ID}\"}}]}, \"limit\": 1, \"with_payload\": true}" 2>/dev/null || echo '{"result":{"points":[]}}')

if echo "$QDRANT_RESULT" | grep -q "$DOC_ID"; then
  pass "Qdrant: Found vector for $DOC_ID"
else
  fail "Qdrant: Vector not found for $DOC_ID"
fi

# Step 5: Report
echo ""
echo "================================"
if [ $FAILURES -eq 0 ]; then
  echo -e "${GREEN}ALL TESTS PASSED${NC}"
  exit 0
else
  echo -e "${RED}${FAILURES} TEST(S) FAILED${NC}"
  exit 1
fi
