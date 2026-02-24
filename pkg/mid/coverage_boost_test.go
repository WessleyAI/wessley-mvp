package mid

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatusWriterWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec}

	// Write without WriteHeader should default to 200
	n, err := sw.Write([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Fatalf("expected 5, got %d", n)
	}
	if sw.status != http.StatusOK {
		t.Fatalf("expected 200, got %d", sw.status)
	}

	// Second Write shouldn't change status
	sw.Write([]byte(" world"))
	if sw.status != http.StatusOK {
		t.Fatalf("status changed unexpectedly")
	}
}

func TestStatusWriterWriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: rec}

	sw.WriteHeader(http.StatusNotFound)
	if sw.status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", sw.status)
	}

	// Second call should not change status
	sw.WriteHeader(http.StatusOK)
	if sw.status != http.StatusNotFound {
		t.Fatalf("status should not change, got %d", sw.status)
	}
}

func TestOTel(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := OTel("test-service")
	wrapped := mw(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestLoggerMiddleware(t *testing.T) {
	log := slog.Default()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	mw := Logger(log)
	wrapped := mw(handler)

	req := httptest.NewRequest("GET", "/hello", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if rec.Body.String() != "ok" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
