# Spec: pkg-infra — Infrastructure Packages

**Branch:** `spec/pkg-infra`  
**Effort:** 3-4 days (1 dev)  
**Priority:** P0 — Foundation, required by api-server, engine-graph, engine-ingest

---

## Scope

Implement three shared infrastructure packages:

1. `pkg/mid/` — HTTP middleware chain
2. `pkg/repo/` — Generic Repository[T] interface + Neo4j implementation  
3. `pkg/natsutil/` — Typed NATS Publish/Subscribe/Request helpers

### Files

```
pkg/mid/chain.go          # Middleware type, Chain, Logger, Recover, CORS
pkg/repo/repo.go          # Repository[T, ID] interface, ListOpts
pkg/repo/neo4j.go         # Neo4jRepo[T, ID] generic implementation
pkg/natsutil/natsutil.go  # Publish[T], Subscribe[T], Request[Req,Resp]
+ tests for each
```

---

## pkg/mid/ — HTTP Middleware

```go
type Middleware func(http.Handler) http.Handler

func Chain(h http.Handler, mw ...Middleware) http.Handler  // left-to-right
func Logger(log *slog.Logger) Middleware     // logs method, path, status, duration
func Recover(log *slog.Logger) Middleware    // catches panics → 500
func CORS(origin string) Middleware          // Allow-Origin, Methods, Headers; 204 on OPTIONS
```

Uses internal `statusWriter` to capture response status code.

## pkg/repo/ — Generic Repository

```go
type Repository[T any, ID comparable] interface {
    Get(ctx context.Context, id ID) (T, error)
    List(ctx context.Context, opts ListOpts) ([]T, error)
    Create(ctx context.Context, entity T) (T, error)
    Update(ctx context.Context, entity T) (T, error)
    Delete(ctx context.Context, id ID) error
}

type ListOpts struct { Offset, Limit int; Filter map[string]any }
```

**Neo4jRepo[T, ID]** — generic Neo4j implementation using functional options:
- Constructor: `NewNeo4jRepo(driver, label, toMap, fromRecord, ...opts)`
- `WithIDKey` option (default "id")
- Cypher: `MATCH (n:{label} {{idKey}: $id})` pattern
- Default `List` limit: 100

## pkg/natsutil/ — NATS Helpers

```go
func Publish[T any](nc *nats.Conn, subject string, v T) error
func Subscribe[T any](nc *nats.Conn, subject string, handler func(T)) (*nats.Subscription, error)
func Request[Req, Resp any](nc *nats.Conn, subject string, req Req) (Resp, error)
```

All use `encoding/json`. Subscribe drops malformed messages. Request uses `nats.DefaultTimeout`.

---

## Acceptance Criteria

- [ ] Middleware chain composes correctly (outermost first)
- [ ] Logger captures status codes; Recover catches panics; CORS handles OPTIONS
- [ ] `Neo4jRepo` implements `Repository` (compile-time check)
- [ ] Functional options for Neo4j config
- [ ] NATS helpers serialize/deserialize JSON
- [ ] Unit tests for middleware (httptest), NATS (mocks or embedded)
- [ ] Integration test pattern for Neo4jRepo

## Dependencies

- `github.com/neo4j/neo4j-go-driver/v5`
- `github.com/nats-io/nats.go`

## Reference

FINAL_ARCHITECTURE.md §6 (mid), §7 (repo), §8.1 (natsutil)


## Feb 15 Refinement: OpenTelemetry Middleware

Add OTel instrumentation to HTTP and gRPC middleware, plus trace ID propagation in NATS messages.

**Impact on this spec:**

### HTTP Middleware (`pkg/mid/`)

```go
// OTel middleware for HTTP — creates spans for each request
func OTel(serviceName string) Middleware {
    return func(next http.Handler) http.Handler {
        return otelhttp.NewHandler(next, serviceName)
    }
}
```

Add `OTel` to the middleware chain in api server setup.

### gRPC Middleware

For ml-worker Go client, use OTel gRPC interceptors:

```go
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

conn, _ := grpc.Dial(addr,
    grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
    grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
)
```

### NATS Trace Propagation (`pkg/natsutil/`)

```go
// Publish injects trace context into NATS message headers
func Publish[T any](nc *nats.Conn, subject string, v T) error {
    msg := &nats.Msg{Subject: subject}
    msg.Data, _ = json.Marshal(v)
    // Inject OTel trace context into NATS headers
    otel.GetTextMapPropagator().Inject(ctx, natsHeaderCarrier(msg))
    return nc.PublishMsg(msg)
}

// Subscribe extracts trace context from NATS message headers
func Subscribe[T any](nc *nats.Conn, subject string, handler func(context.Context, T)) (*nats.Subscription, error) {
    // Extract OTel trace context from NATS headers
    // ctx := otel.GetTextMapPropagator().Extract(context.Background(), natsHeaderCarrier(msg))
    // handler(ctx, parsed)
}
```

Note: `Publish` and `Subscribe` signatures gain a `context.Context` parameter for trace propagation.

### Additional acceptance criteria
- [ ] `mid.OTel()` middleware creates HTTP spans via `otelhttp`
- [ ] gRPC client uses OTel interceptors for ml-worker calls
- [ ] NATS Publish/Subscribe propagate trace IDs via message headers
- [ ] Trace IDs flow end-to-end: HTTP → NATS → subscriber → gRPC
- [ ] New dependencies: `go.opentelemetry.io/otel`, `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`, `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`
