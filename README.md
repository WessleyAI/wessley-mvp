# Wessley.ai MVP

RAG chatbot for vehicle electrical systems. Ask questions about any vehicle's wiring, get AI-powered answers grounded in real automotive knowledge.

## Architecture

3 services, 10 patterns, functional Go pipelines.

| Service | Language | Role |
|---------|----------|------|
| **api** | Go | HTTP API, auth, chat endpoints |
| **engine** | Go | Scraper, knowledge graph, ingestion, semantic search |
| **ml-worker** | Python | Self-hosted LLM (Ollama), embeddings, vector search |
| **web** | Next.js | Minimal chat frontend |

## Stack

- **Backend:** Go 1.22+
- **ML:** Python + Ollama (Mistral/Llama 3) + local embeddings
- **Data:** Neo4j, Qdrant, Redis, NATS JetStream
- **Frontend:** Next.js 15
- **Infra:** Docker Compose → K8s

## Development

```bash
docker compose up -d    # Start infra (Neo4j, Qdrant, Redis, NATS, Ollama)
make run-engine         # Start engine service
make run-api            # Start API service
make run-web            # Start Next.js frontend
```

## License

MIT
