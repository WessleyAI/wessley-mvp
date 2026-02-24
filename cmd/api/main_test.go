package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/health", nil)
	handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", resp["status"])
	}
}

func TestChatEndpoint_EmptyQuestion(t *testing.T) {
	handler := handleChat(nil, nil)
	body := `{"question":""}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewBufferString(body))
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestChatEndpoint_InvalidJSON(t *testing.T) {
	handler := handleChat(nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/chat", bytes.NewBufferString("not json"))
	handler(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := loadConfig()
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.CORSOrigin != "*" {
		t.Fatalf("expected default CORS *, got %s", cfg.CORSOrigin)
	}
	if cfg.Collection != "wessley" {
		t.Fatalf("expected default collection wessley, got %s", cfg.Collection)
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("TEST_ENV_VAR_XYZ", "custom")
	if v := envOr("TEST_ENV_VAR_XYZ", "default"); v != "custom" {
		t.Fatalf("expected custom, got %s", v)
	}
	if v := envOr("NONEXISTENT_VAR_ABC", "fallback"); v != "fallback" {
		t.Fatalf("expected fallback, got %s", v)
	}
}
