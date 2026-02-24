.PHONY: proto up down logs seed dev run-wessley run-web test

proto: ## Generate protobuf Go + Python stubs
	cd proto && buf lint
	cd proto && buf generate

up:
	docker compose -f deploy/docker-compose.yml up -d

down:
	docker compose -f deploy/docker-compose.yml down

logs:
	docker compose -f deploy/docker-compose.yml logs -f

seed:
	./deploy/scripts/seed-ollama.sh

dev:
	docker compose -f deploy/docker-compose.yml -f deploy/docker-compose.dev.yml up -d

run-wessley:
	go run ./cmd/wessley

run-web:
	cd web && npm run dev

test:
	go test ./...
