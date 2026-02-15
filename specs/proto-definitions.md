# Spec: proto-definitions — gRPC Service Definitions

**Branch:** `spec/proto-definitions`
**Effort:** 1 day
**Priority:** P1 — Phase 1

---

## Scope

Protocol Buffer definitions for the ml-worker gRPC contract. Shared between Go (client) and Python (server). Managed with Buf.

**Only two services remain: ChatService + EmbedService.** SearchService has been removed — search is now handled by engine/semantic (Go, direct Qdrant access).

### Files

```
proto/
├── buf.yaml              # Buf project config
├── buf.gen.yaml          # Code generation config (Go + Python)
├── ml/v1/
│   ├── chat.proto        # ChatService
│   └── embed.proto       # EmbedService
ml/proto/                 # Generated Go code (output)
ml-worker/proto/          # Generated Python code (output)
```

## Proto Definitions

### chat.proto

```protobuf
syntax = "proto3";
package wessley.ml.v1;
option go_package = "wessley-mvp/ml/proto;mlpb";

service ChatService {
    rpc Chat(ChatRequest) returns (ChatResponse);
    rpc ChatStream(ChatRequest) returns (stream ChatChunk);
}

message ChatRequest {
    string message = 1;
    repeated string context = 2;
    string system_prompt = 3;
    float temperature = 4;
    string model = 5;           // "mistral", "llama3", etc.
    int32 max_tokens = 6;
}

message ChatResponse {
    string reply = 1;
    int32 tokens_used = 2;
    string model = 3;
}

message ChatChunk {
    string text = 1;
    bool done = 2;
}
```

### embed.proto

```protobuf
syntax = "proto3";
package wessley.ml.v1;
option go_package = "wessley-mvp/ml/proto;mlpb";

service EmbedService {
    rpc Embed(EmbedRequest) returns (EmbedResponse);
    rpc EmbedBatch(EmbedBatchRequest) returns (EmbedBatchResponse);
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

### search.proto — REMOVED

SearchService has been removed from ml-worker proto. Vector search is now owned by `engine/semantic` which accesses Qdrant directly via Go client. No gRPC proxy needed.

## Buf Config

```yaml
# buf.yaml
version: v2
modules:
  - path: proto
lint:
  use: [DEFAULT]
breaking:
  use: [FILE]
```

```yaml
# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: ml/proto
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: ml/proto
    opt: paths=source_relative
  - remote: buf.build/protocolbuffers/python
    out: ml-worker/proto
  - remote: buf.build/grpc/python
    out: ml-worker/proto
```

## Acceptance Criteria

- [ ] 2 proto files compile with `buf build` (chat + embed only)
- [ ] Go code generates to `ml/proto/`
- [ ] Python code generates to `ml-worker/proto/`
- [ ] Lint passes with `buf lint`
- [ ] Breaking change detection with `buf breaking`
- [ ] Makefile target: `make proto`
- [ ] No SearchService — removed, search owned by engine/semantic

## Dependencies

- `buf` CLI installed
- protoc-gen-go, protoc-gen-go-grpc

## Reference

- FINAL_ARCHITECTURE.md §8.1
- ml-worker spec for service definitions


## Feb 15 Refinement: Monolith for MVP

With the monolith decision, proto definitions are **only used for wessley ↔ ml-worker communication**. There is no inter-service gRPC between api and engine — they are merged into a single binary (`cmd/wessley/`).

**Impact on this spec:**
- Proto files define the contract between `wessley` (Go monolith) and `ml-worker` (Python) only
- No proto definitions needed for api↔engine communication (direct Go imports instead)
- Only 2 services exist: `wessley` + `ml-worker`
- `buf generate` outputs: Go stubs into `ml/proto/` (used by wessley binary), Python stubs into `ml-worker/proto/`
- All internal engine packages (rag, semantic, graph, ingest, scraper, domain) are imported directly via Go imports within the wessley binary
