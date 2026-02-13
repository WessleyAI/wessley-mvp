# Spec: deploy-compose — Docker Compose

**Branch:** `spec/deploy-compose`
**Effort:** 1-2 days
**Priority:** P1 — Phase 1

---

## Scope

Docker Compose configuration for the full MVP stack: all 3 services + infrastructure.

### Files

```
deploy/
├── docker-compose.yml
├── docker-compose.dev.yml    # Dev overrides (hot reload, volumes)
├── .env.example
└── scripts/
    ├── seed-ollama.sh        # Pull Mistral model on first run
    └── wait-for-it.sh        # Health check waiter
Dockerfile.api                # Go API service
Dockerfile.engine             # Go Engine service
Dockerfile.ml-worker          # Python ML worker
Makefile                      # Convenience commands
```

## Services

```yaml
services:
  # --- App Services ---
  api:
    build: { dockerfile: Dockerfile.api }
    ports: ["8080:8080"]
    env: [NATS_URL, ML_WORKER_ADDR, LOG_LEVEL]
    depends_on: [nats, ml-worker]

  engine:
    build: { dockerfile: Dockerfile.engine }
    env: [NATS_URL, NEO4J_URI, NEO4J_AUTH, QDRANT_ADDR, REDIS_ADDR, ML_WORKER_ADDR]
    depends_on: [nats, neo4j, qdrant, redis, ml-worker]

  ml-worker:
    build: { dockerfile: Dockerfile.ml-worker }
    ports: ["50051:50051"]
    env: [OLLAMA_HOST, QDRANT_ADDR, EMBEDDING_MODEL]
    depends_on: [ollama, qdrant]

  web:
    build: { context: web/ }
    ports: ["3000:3000"]
    env: [API_URL=http://api:8080]
    depends_on: [api]

  # --- Infrastructure ---
  neo4j:
    image: neo4j:5-community
    ports: ["7474:7474", "7687:7687"]
    environment: [NEO4J_AUTH=neo4j/password, "NEO4J_PLUGINS=[\"apoc\"]"]
    volumes: [neo4j_data:/data]
    healthcheck: { test: ["CMD", "wget", "-qO-", "http://localhost:7474"], interval: 10s }

  qdrant:
    image: qdrant/qdrant:latest
    ports: ["6333:6333", "6334:6334"]
    volumes: [qdrant_data:/qdrant/storage]
    healthcheck: { test: ["CMD", "curl", "-f", "http://localhost:6333/healthz"], interval: 10s }

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    volumes: [redis_data:/data]

  nats:
    image: nats:2-alpine
    ports: ["4222:4222", "8222:8222"]
    command: ["--jetstream", "--store_dir=/data"]
    volumes: [nats_data:/data]

  ollama:
    image: ollama/ollama:latest
    ports: ["11434:11434"]
    volumes: [ollama_data:/root/.ollama]
    # Pull model on first run via seed script

volumes:
  neo4j_data:
  qdrant_data:
  redis_data:
  nats_data:
  ollama_data:
```

## Makefile

```makefile
up:           docker compose up -d
down:         docker compose down
logs:         docker compose logs -f
seed:         docker exec ollama ollama pull mistral
run-api:      go run ./cmd/api
run-engine:   go run ./cmd/engine
run-web:      cd web && npm run dev
test:         go test ./...
```

## Acceptance Criteria

- [ ] `docker compose up` starts all services
- [ ] Health checks on all infra services
- [ ] Ollama model seeding script
- [ ] .env.example with all required vars
- [ ] Dev overrides with hot reload for Go (air) and Next.js
- [ ] Multi-stage Dockerfiles (small images)
- [ ] Makefile for common commands
- [ ] Networking: all services on same Docker network
- [ ] Volumes for persistent data

## Dependencies

- Docker, Docker Compose v2

## Reference

- FINAL_ARCHITECTURE.md §2 (service architecture)
