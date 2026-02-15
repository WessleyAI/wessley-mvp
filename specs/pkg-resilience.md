# Spec: pkg-resilience — Circuit Breaker + Rate Limiter

**Branch:** `spec/pkg-resilience`
**Effort:** 1-2 days
**Priority:** P1 — Phase 2

---

## Scope

Shared resilience primitives: circuit breaker for ml-worker calls, token bucket rate limiter for scrapers. Thin wrappers over proven libraries.

### Files

```
pkg/resilience/breaker.go       # Circuit breaker wrapper
pkg/resilience/limiter.go       # Token bucket rate limiter
pkg/resilience/breaker_test.go
pkg/resilience/limiter_test.go
```

## Circuit Breaker

```go
// pkg/resilience/breaker.go

// Thin wrapper around sony/gobreaker for ml-worker gRPC calls.

type Breaker struct {
    cb *gobreaker.CircuitBreaker
}

type BreakerConfig struct {
    Name          string
    MaxRequests   uint32        // max requests in half-open state (default: 3)
    Interval      time.Duration // cyclic reset interval (default: 60s)
    Timeout       time.Duration // time in open state before half-open (default: 30s)
    FailThreshold uint32        // failures to trip (default: 5)
}

func NewBreaker(cfg BreakerConfig) *Breaker

// Execute runs fn through the circuit breaker
func (b *Breaker) Execute(fn func() (any, error)) (any, error)

// State returns the current breaker state (closed/half-open/open)
func (b *Breaker) State() gobreaker.State
```

### Usage

```go
mlBreaker := resilience.NewBreaker(resilience.BreakerConfig{
    Name: "ml-worker",
    FailThreshold: 5,
    Timeout: 30 * time.Second,
})

result, err := mlBreaker.Execute(func() (any, error) {
    return mlClient.Embed(ctx, text)
})
```

## Rate Limiter

```go
// pkg/resilience/limiter.go

// Token bucket rate limiter, shared by all scrapers.
// Configurable per-source (e.g., Reddit: 60 req/min, YouTube: 100 req/day).

type Limiter struct {
    limiter *rate.Limiter
}

type LimiterConfig struct {
    Rate  float64       // tokens per second
    Burst int           // max burst size
}

func NewLimiter(cfg LimiterConfig) *Limiter

// Wait blocks until a token is available or ctx is cancelled
func (l *Limiter) Wait(ctx context.Context) error

// Allow returns true if a token is available (non-blocking)
func (l *Limiter) Allow() bool
```

### Usage

```go
// Shared limiter for Reddit (60 req/min = 1 req/sec)
redditLimiter := resilience.NewLimiter(resilience.LimiterConfig{
    Rate:  1.0,
    Burst: 5,
})

// In scraper loop:
if err := redditLimiter.Wait(ctx); err != nil {
    return // context cancelled
}
// ... make request
```

## Acceptance Criteria

- [ ] Circuit breaker wraps sony/gobreaker with sensible defaults
- [ ] Breaker trips after configurable failure threshold
- [ ] Breaker auto-recovers via half-open state
- [ ] Token bucket rate limiter with configurable rate + burst
- [ ] Limiter supports blocking (Wait) and non-blocking (Allow)
- [ ] Shared across scrapers (not per-instance)
- [ ] Unit tests for both breaker and limiter

## Dependencies

- `github.com/sony/gobreaker`
- `golang.org/x/time/rate`

## Reference

- FINAL_ARCHITECTURE.md §8.2 (scraper rate limiting)
- FINAL_ARCHITECTURE.md §8.5 (ml-worker resilience)
