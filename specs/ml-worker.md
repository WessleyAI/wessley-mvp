# Spec: ml-worker — Python gRPC Server with Ollama

**Branch:** `spec/ml-worker`
**Effort:** 2-3 days
**Priority:** P1 — Phase 3

---

## Scope

Python gRPC server that wraps Ollama (self-hosted LLM) and a local embedding model. Zero API costs. Provides **only** chat completion and embedding generation. Does NOT access Qdrant — search is handled by engine/semantic.

### Files

```
ml-worker/
├── proto/ml.proto          # gRPC definitions (shared with Go)
├── ml_worker/
│   ├── server.py           # gRPC server entrypoint
│   ├── chat.py             # Ollama chat completion
│   ├── embed.py            # Local embedding model
│   └── config.py           # Environment config
├── requirements.txt
├── Dockerfile
└── tests/
    └── test_server.py
```

## LLM Stack

| Component | Model | Purpose |
|-----------|-------|---------|
| **Chat** | Mistral 7B or Llama 3 8B via Ollama | RAG response generation |
| **Embeddings** | all-MiniLM-L6-v2 (sentence-transformers) | 384-dim embeddings, runs on CPU |

## gRPC Services

Only two services — ChatService and EmbedService. SearchService has been removed (search is owned by engine/semantic).

```protobuf
service ChatService {
    rpc Chat(ChatRequest) returns (ChatResponse);
    rpc ChatStream(ChatRequest) returns (stream ChatChunk);
}

service EmbedService {
    rpc Embed(EmbedRequest) returns (EmbedResponse);
    rpc EmbedBatch(EmbedBatchRequest) returns (EmbedBatchResponse);
}

message ChatRequest {
    string message = 1;
    repeated string context = 2;     // RAG context chunks
    string system_prompt = 3;
    float temperature = 4;
}

message ChatResponse {
    string reply = 1;
    int32 tokens_used = 2;
}

message ChatChunk {
    string text = 1;
    bool done = 2;
}

message EmbedRequest {
    string text = 1;
}

message EmbedResponse {
    repeated float values = 1;
    int32 dimensions = 2;
}

message EmbedBatchRequest {
    repeated string texts = 1;
}

message EmbedBatchResponse {
    repeated EmbedResponse embeddings = 1;
}
```

## Ollama Integration

```python
import ollama

def chat(message: str, context: list[str], model: str = "mistral") -> str:
    system = build_system_prompt()
    ctx = "\n---\n".join(context)
    prompt = f"Context:\n{ctx}\n\nQuestion: {message}"
    response = ollama.chat(model=model, messages=[
        {"role": "system", "content": system},
        {"role": "user", "content": prompt},
    ])
    return response["message"]["content"]
```

## Embedding

```python
from sentence_transformers import SentenceTransformer

model = SentenceTransformer("all-MiniLM-L6-v2")  # 384 dims, ~80MB, CPU-friendly

def embed(text: str) -> list[float]:
    return model.encode(text).tolist()

def embed_batch(texts: list[str]) -> list[list[float]]:
    return model.encode(texts).tolist()
```

## System Prompt

```
You are Wessley, an AI assistant specializing in vehicle electrical systems.
You answer questions about wiring, components, diagnostics, and repairs.
Use the provided context to give accurate, specific answers.
If the context doesn't contain relevant information, say so honestly.
Always mention the vehicle year/make/model when relevant.
```

## Acceptance Criteria

- [ ] gRPC server on port 50051
- [ ] Chat completion via Ollama (Mistral 7B default)
- [ ] Streaming chat responses
- [ ] Local embeddings via sentence-transformers (all-MiniLM-L6-v2)
- [ ] Batch embedding support
- [ ] NO Qdrant access — search removed, owned by engine/semantic
- [ ] Health check endpoint
- [ ] Graceful shutdown
- [ ] Dockerfile with Python 3.11 slim
- [ ] < 2GB container image
- [ ] Works on CPU (no GPU required for MVP)
- [ ] Unit tests

## Dependencies

- `ollama` Python package + Ollama server (Docker)
- `sentence-transformers`
- `grpcio`, `protobuf`

## Reference

- FINAL_ARCHITECTURE.md §8.5, §8.6


## Feb 15 Refinement: Separate Embed vs Chat Workers

ml-worker supports a `--mode` flag to run as embed-only, chat-only, or both. This enables independent scaling of embedding and chat workloads.

### Mode Flag

```
ml-worker --mode=embed   # Only EmbedService, subscribes to ml.embed.*
ml-worker --mode=chat    # Only ChatService, subscribes to ml.chat.*
ml-worker --mode=all     # Both services (MVP default)
```

### NATS Subjects

| Mode | NATS Subject | Ack Timeout | Rationale |
|------|-------------|-------------|-----------|
| embed | `ml.embed.*` | 30s | Embeddings are fast (CPU, batched) |
| chat | `ml.chat.*` | 120s | LLM generation is slow (GPU-bound) |
| all | `ml.embed.*` + `ml.chat.*` | per-subject | Single process handles both |

### Implementation

```python
# ml-worker/src/server.py
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("--mode", choices=["embed", "chat", "all"], default="all")
args = parser.parse_args()

if args.mode in ("embed", "all"):
    server.add_insecure_port(...)  # EmbedService
    # Subscribe to ml.embed.* with 30s ack timeout

if args.mode in ("chat", "all"):
    server.add_insecure_port(...)  # ChatService
    # Subscribe to ml.chat.* with 120s ack timeout
```

### Scaling Strategy

- **MVP:** Single `ml-worker --mode=all` process handles everything
- **Scale:** Separate `ml-worker --mode=embed` (CPU nodes) and `ml-worker --mode=chat` (GPU nodes)
- NATS subject separation ensures messages route to the correct worker type

### Additional acceptance criteria
- [ ] `--mode=embed|chat|all` flag controls which services start
- [ ] Different NATS subjects: `ml.embed.*` vs `ml.chat.*`
- [ ] Different ack timeouts: 30s for embed, 120s for chat
- [ ] MVP runs `--mode=all` (single process, both services)
- [ ] Can scale by running separate embed and chat workers
