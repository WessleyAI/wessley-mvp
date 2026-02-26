package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	r := New()
	c := r.Counter("test_total", "A test counter")
	if c.Value() != 0 {
		t.Fatalf("expected 0, got %d", c.Value())
	}
	c.Inc()
	c.Inc()
	c.Add(5)
	if c.Value() != 7 {
		t.Fatalf("expected 7, got %d", c.Value())
	}
	// Same name returns same counter
	c2 := r.Counter("test_total", "")
	if c2 != c {
		t.Fatal("expected same counter instance")
	}
}

func TestGauge(t *testing.T) {
	r := New()
	g := r.Gauge("test_gauge", "A test gauge")
	g.Set(42)
	if g.Value() != 42 {
		t.Fatalf("expected 42, got %d", g.Value())
	}
	g.Inc()
	g.Inc()
	g.Dec()
	if g.Value() != 43 {
		t.Fatalf("expected 43, got %d", g.Value())
	}
}

func TestHistogram(t *testing.T) {
	r := New()
	h := r.Histogram("test_duration_seconds", "A test histogram", []float64{0.1, 0.5, 1.0})
	h.Observe(0.05)
	h.Observe(0.3)
	h.Observe(0.8)
	h.Observe(2.0)

	buckets, counts, sum, count := h.snapshot()
	if count != 4 {
		t.Fatalf("expected count 4, got %d", count)
	}
	if len(buckets) != 3 {
		t.Fatalf("expected 3 buckets, got %d", len(buckets))
	}
	// 0.05 <= 0.1
	if counts[0] != 1 {
		t.Fatalf("bucket 0.1: expected 1, got %d", counts[0])
	}
	// 0.3 falls into bucket 0.5
	if counts[1] != 1 {
		t.Fatalf("bucket 0.5: expected 1, got %d", counts[1])
	}
	// 0.8 falls into bucket 1.0
	if counts[2] != 1 {
		t.Fatalf("bucket 1.0: expected 1, got %d", counts[2])
	}
	expectedSum := 0.05 + 0.3 + 0.8 + 2.0
	if sum != expectedSum {
		t.Fatalf("expected sum %f, got %f", expectedSum, sum)
	}
}

func TestHistogramSince(t *testing.T) {
	r := New()
	h := r.Histogram("latency", "", nil)
	start := time.Now().Add(-100 * time.Millisecond)
	h.Since(start)
	_, _, _, count := h.snapshot()
	if count != 1 {
		t.Fatal("expected 1 observation")
	}
}

func TestWithLabels(t *testing.T) {
	got := WithLabels("foo_total", "source", "reddit", "type", "post")
	want := `foo_total{source="reddit",type="post"}`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	// No labels
	if WithLabels("bar") != "bar" {
		t.Fatal("no labels should return name unchanged")
	}
}

func TestRender(t *testing.T) {
	r := New()
	r.Counter("requests_total", "Total requests").Add(10)
	r.Counter(WithLabels("requests_total", "method", "GET"), "").Add(7)
	r.Counter(WithLabels("requests_total", "method", "POST"), "").Add(3)
	r.Gauge("active_connections", "Active conns").Set(5)
	h := r.Histogram("request_duration_seconds", "Request latency", []float64{0.1, 0.5, 1.0})
	h.Observe(0.05)
	h.Observe(0.3)

	out := r.Render()

	// Check TYPE headers
	if !strings.Contains(out, "# TYPE requests_total counter") {
		t.Error("missing TYPE for counter")
	}
	if !strings.Contains(out, "# TYPE active_connections gauge") {
		t.Error("missing TYPE for gauge")
	}
	if !strings.Contains(out, "# TYPE request_duration_seconds histogram") {
		t.Error("missing TYPE for histogram")
	}
	// Check values
	if !strings.Contains(out, "requests_total 10") {
		t.Error("missing plain counter value")
	}
	if !strings.Contains(out, `requests_total{method="GET"} 7`) {
		t.Error("missing labeled counter")
	}
	if !strings.Contains(out, "active_connections 5") {
		t.Error("missing gauge value")
	}
	// Check histogram buckets
	if !strings.Contains(out, `request_duration_seconds_bucket{le="0.1"} 1`) {
		t.Errorf("missing histogram bucket 0.1, got:\n%s", out)
	}
	if !strings.Contains(out, `request_duration_seconds_bucket{le="+Inf"} 2`) {
		t.Error("missing +Inf bucket")
	}
	if !strings.Contains(out, "request_duration_seconds_count 2") {
		t.Error("missing histogram count")
	}
}

func TestHandler(t *testing.T) {
	r := New()
	r.Counter("test_total", "test").Inc()

	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	r.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/plain") {
		t.Fatalf("unexpected content type: %s", ct)
	}
	if !strings.Contains(rec.Body.String(), "test_total 1") {
		t.Error("missing metric in handler output")
	}
}

func TestMetricBaseName(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"foo_total", "foo_total"},
		{`foo_total{k="v"}`, "foo_total"},
		{`foo{a="1",b="2"}`, "foo"},
	}
	for _, tt := range tests {
		got := metricBaseName(tt.in)
		if got != tt.want {
			t.Errorf("metricBaseName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestGaugeFloat(t *testing.T) {
	r := New()
	g := r.Gauge("float_gauge", "")
	g.SetFloat(3.14)
	// Note: Render will show the int64 bits, but FloatValue works correctly
	if g.FloatValue() != 3.14 {
		t.Fatalf("expected 3.14, got %f", g.FloatValue())
	}
}
