# Spec: proto-definitions — gRPC Service Definitions

**Branch:** `spec/proto-definitions`
**Effort:** 1 day
**Priority:** P1 — Phase 1

---

## Scope

Protocol Buffer definitions for the ml-worker gRPC contract. Shared between Go (client) and Python (server). Managed with Buf.

### Files

```
proto/
├── buf.yaml              # Buf project config
├── buf.gen.yaml          # Code generation config (Go + Python)
├── ml/v1/
│   ├── chat.proto        # ChatService
│   ├── embed.proto       # EmbedService
│   └── search.proto      # SearchService
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

### search.proto

```protobuf
syntax = "proto3";
package wessley.ml.v1;
option go_package = "wessley-mvp/ml/proto;mlpb";

service SearchService {
    rpc Search(SearchRequest) returns (SearchResponse);
}

message SearchRequest {
    string query = 1;
    int32 top_k = 2;
    map<string, string> filters = 3;
}

message SearchResult {
    string id = 1;
    float score = 2;
    string content = 3;
    string doc_id = 4;
    string source = 5;
    map<string, string> metadata = 6;
}

message SearchResponse {
    repeated SearchResult results = 1;
}
```

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

- [ ] All 3 proto files compile with `buf build`
- [ ] Go code generates to `ml/proto/`
- [ ] Python code generates to `ml-worker/proto/`
- [ ] Lint passes with `buf lint`
- [ ] Breaking change detection with `buf breaking`
- [ ] Makefile target: `make proto`

## Dependencies

- `buf` CLI installed
- protoc-gen-go, protoc-gen-go-grpc

## Reference

- FINAL_ARCHITECTURE.md §8.1 (NATS helpers reference proto patterns)
- ml-worker spec for full message definitions
