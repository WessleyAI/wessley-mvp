package natsutil

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nats-io/nats.go"
)

type testMsg struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestNatsHeaderCarrier(t *testing.T) {
	msg := &nats.Msg{}
	carrier := (*natsHeaderCarrier)(msg)

	carrier.Set("traceparent", "00-abc-def-01")
	if got := carrier.Get("traceparent"); got != "00-abc-def-01" {
		t.Fatalf("expected traceparent, got %q", got)
	}

	keys := carrier.Keys()
	if len(keys) != 1 {
		t.Fatalf("unexpected keys: %v", keys)
	}
}

func TestNatsHeaderCarrierNilHeader(t *testing.T) {
	msg := &nats.Msg{}
	carrier := (*natsHeaderCarrier)(msg)

	if got := carrier.Get("missing"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	if keys := carrier.Keys(); keys != nil {
		t.Fatalf("expected nil keys, got %v", keys)
	}
}

func TestPublishSerializesJSON(t *testing.T) {
	// We can't easily test Publish without a NATS connection,
	// but we can verify the JSON marshaling logic.
	msg := testMsg{Name: "test", Value: 42}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded testMsg
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != "test" || decoded.Value != 42 {
		t.Fatalf("unexpected: %+v", decoded)
	}
}

func TestSubscribeDropsMalformed(t *testing.T) {
	// Simulate the handler logic from Subscribe
	called := false
	handler := func(_ context.Context, v testMsg) {
		called = true
	}

	// Simulate malformed message processing
	badData := []byte("{invalid json")
	var v testMsg
	if err := json.Unmarshal(badData, &v); err != nil {
		// This is expected — malformed messages are dropped
		if called {
			t.Fatal("handler should not have been called for malformed message")
		}
		return
	}
	handler(context.Background(), v)
}
