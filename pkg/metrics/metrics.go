// Package metrics provides a lightweight Prometheus-compatible metrics
// registry using only the standard library. It supports counters, gauges,
// and histograms with optional labels, and exposes them via an HTTP /metrics
// endpoint in the Prometheus text exposition format.
package metrics

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultBuckets are the default histogram buckets (in seconds).
var DefaultBuckets = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60}

// Counter is a monotonically increasing counter.
type Counter struct{ val atomic.Int64 }

func (c *Counter) Inc()        { c.val.Add(1) }
func (c *Counter) Add(n int64) { c.val.Add(n) }
func (c *Counter) Value() int64 { return c.val.Load() }

// Gauge can go up and down.
type Gauge struct{ val atomic.Int64 }

func (g *Gauge) Set(n int64)   { g.val.Store(n) }
func (g *Gauge) Inc()          { g.val.Add(1) }
func (g *Gauge) Dec()          { g.val.Add(-1) }
func (g *Gauge) Value() int64  { return g.val.Load() }

// SetFloat stores a float64 as int64 bits.
func (g *Gauge) SetFloat(f float64) { g.val.Store(int64(math.Float64bits(f))) }

// FloatValue returns the gauge value interpreted as float64 bits.
func (g *Gauge) FloatValue() float64 { return math.Float64frombits(uint64(g.val.Load())) }

// Histogram tracks the distribution of observed values using fixed buckets.
type Histogram struct {
	mu      sync.Mutex
	buckets []float64
	counts  []uint64 // one per bucket
	sum     float64
	count   uint64
}

func newHistogram(buckets []float64) *Histogram {
	b := make([]float64, len(buckets))
	copy(b, buckets)
	sort.Float64s(b)
	return &Histogram{buckets: b, counts: make([]uint64, len(b))}
}

// Observe records a value.
func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	h.sum += v
	h.count++
	// Find the first bucket where v <= b and increment only that one.
	// Render will accumulate cumulatively.
	for i, b := range h.buckets {
		if v <= b {
			h.counts[i]++
			break
		}
	}
	h.mu.Unlock()
}

// Since is a convenience to observe duration since t.
func (h *Histogram) Since(t time.Time) {
	h.Observe(time.Since(t).Seconds())
}

// snapshot returns a copy of the histogram state.
func (h *Histogram) snapshot() ([]float64, []uint64, float64, uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c := make([]uint64, len(h.counts))
	copy(c, h.counts)
	return h.buckets, c, h.sum, h.count
}

// Registry holds named metrics.
type Registry struct {
	mu         sync.RWMutex
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	help       map[string]string
	types      map[string]string // "counter", "gauge", "histogram"
	order      []string          // insertion order
}

// New creates a new Registry.
func New() *Registry {
	return &Registry{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		help:       make(map[string]string),
		types:      make(map[string]string),
	}
}

func (r *Registry) track(name, typ, help string) {
	if _, ok := r.types[name]; !ok {
		r.order = append(r.order, name)
	}
	r.types[name] = typ
	if help != "" {
		r.help[name] = help
	}
}

// Counter returns (or creates) a counter. Label pairs are baked into the name
// as name{k="v",...} so each label combo is a distinct metric line.
func (r *Registry) Counter(name, help string) *Counter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c, ok := r.counters[name]; ok {
		return c
	}
	c := &Counter{}
	r.counters[name] = c
	baseName := metricBaseName(name)
	r.track(baseName, "counter", help)
	return c
}

// Gauge returns (or creates) a gauge.
func (r *Registry) Gauge(name, help string) *Gauge {
	r.mu.Lock()
	defer r.mu.Unlock()
	if g, ok := r.gauges[name]; ok {
		return g
	}
	g := &Gauge{}
	r.gauges[name] = g
	baseName := metricBaseName(name)
	r.track(baseName, "gauge", help)
	return g
}

// Histogram returns (or creates) a histogram.
func (r *Registry) Histogram(name, help string, buckets []float64) *Histogram {
	if buckets == nil {
		buckets = DefaultBuckets
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if h, ok := r.histograms[name]; ok {
		return h
	}
	h := newHistogram(buckets)
	r.histograms[name] = h
	baseName := metricBaseName(name)
	r.track(baseName, "histogram", help)
	return h
}

// WithLabels returns a metric name with labels appended, e.g.
// WithLabels("foo", "k", "v") => `foo{k="v"}`
func WithLabels(name string, kvs ...string) string {
	if len(kvs) == 0 || len(kvs)%2 != 0 {
		return name
	}
	var b strings.Builder
	b.WriteString(name)
	b.WriteByte('{')
	for i := 0; i < len(kvs); i += 2 {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(kvs[i])
		b.WriteString(`="`)
		b.WriteString(kvs[i+1])
		b.WriteByte('"')
	}
	b.WriteByte('}')
	return b.String()
}

// metricBaseName strips labels from a metric name.
func metricBaseName(name string) string {
	if idx := strings.IndexByte(name, '{'); idx != -1 {
		return name[:idx]
	}
	return name
}

// Render returns the Prometheus text exposition format output.
func (r *Registry) Render() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var b strings.Builder

	// Collect all metric names (base names from order)
	rendered := make(map[string]bool)

	for _, baseName := range r.order {
		if rendered[baseName] {
			continue
		}
		rendered[baseName] = true

		typ := r.types[baseName]
		if h, ok := r.help[baseName]; ok {
			fmt.Fprintf(&b, "# HELP %s %s\n", baseName, h)
		}
		fmt.Fprintf(&b, "# TYPE %s %s\n", baseName, typ)

		switch typ {
		case "counter":
			// Find all counters with this base name
			names := r.sortedKeysPrefix(r.counterNames(), baseName)
			for _, n := range names {
				c := r.counters[n]
				fmt.Fprintf(&b, "%s %d\n", n, c.Value())
			}
		case "gauge":
			names := r.sortedKeysPrefix(r.gaugeNames(), baseName)
			for _, n := range names {
				g := r.gauges[n]
				fmt.Fprintf(&b, "%s %d\n", n, g.Value())
			}
		case "histogram":
			names := r.sortedKeysPrefix(r.histogramNames(), baseName)
			for _, n := range names {
				h := r.histograms[n]
				buckets, counts, sum, count := h.snapshot()
				labels := extractLabels(n)
				cumulative := uint64(0)
				for i, bk := range buckets {
					cumulative += counts[i]
					fmt.Fprintf(&b, "%s_bucket{le=\"%g\"%s} %d\n", baseName, bk, labels, cumulative)
				}
				fmt.Fprintf(&b, "%s_bucket{le=\"+Inf\"%s} %d\n", baseName, labels, count)
				fmt.Fprintf(&b, "%s_sum%s %g\n", baseName, wrapLabels(labels), sum)
				fmt.Fprintf(&b, "%s_count%s %d\n", baseName, wrapLabels(labels), count)
			}
		}
	}
	return b.String()
}

func (r *Registry) counterNames() []string {
	names := make([]string, 0, len(r.counters))
	for n := range r.counters {
		names = append(names, n)
	}
	return names
}

func (r *Registry) gaugeNames() []string {
	names := make([]string, 0, len(r.gauges))
	for n := range r.gauges {
		names = append(names, n)
	}
	return names
}

func (r *Registry) histogramNames() []string {
	names := make([]string, 0, len(r.histograms))
	for n := range r.histograms {
		names = append(names, n)
	}
	return names
}

func (r *Registry) sortedKeysPrefix(names []string, prefix string) []string {
	var out []string
	for _, n := range names {
		if metricBaseName(n) == prefix {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// extractLabels returns the label portion from a metric name like `foo{k="v"}` as `,k="v"`.
func extractLabels(name string) string {
	idx := strings.IndexByte(name, '{')
	if idx == -1 {
		return ""
	}
	// Return the inner part prefixed with comma for bucket label injection
	inner := name[idx+1 : len(name)-1]
	if inner == "" {
		return ""
	}
	return "," + inner
}

// wrapLabels wraps extracted label string (like `,k="v"`) into `{k="v"}` or empty string.
func wrapLabels(labels string) string {
	if labels == "" {
		return ""
	}
	// labels starts with comma, strip it and wrap
	return "{" + labels[1:] + "}"
}

// Handler returns an http.Handler that serves /metrics.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Write([]byte(r.Render()))
	})
}

// Serve starts an HTTP server on the given port serving /metrics.
func (r *Registry) Serve(port int) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", r.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok\n"))
	})
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

// ServeAsync starts the metrics server in a goroutine. Errors are logged.
func (r *Registry) ServeAsync(port int) {
	go func() {
		if err := r.Serve(port); err != nil {
			fmt.Printf("metrics server error on port %d: %v\n", port, err)
		}
	}()
}
