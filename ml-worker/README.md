# ML Worker

gRPC server providing LLM chat (via Ollama) and embedding (via sentence-transformers) services.

## Services

- **ChatService** — `Chat` (unary) and `ChatStream` (server-streaming) via Ollama
- **EmbedService** — `Embed` (single) and `EmbedBatch` (batch) via sentence-transformers
- **Health** — standard gRPC health checking

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama API base URL |
| `CHAT_MODEL` | `mistral` | Default chat model |
| `EMBEDDING_MODEL_NAME` | `all-MiniLM-L6-v2` | sentence-transformers model |
| `LISTEN_ADDR` | `[::]:50051` | gRPC listen address |
| `LOG_LEVEL` | `INFO` | Logging level |

## Run locally

```bash
pip install -r requirements.txt
python -m ml_worker.server
```
