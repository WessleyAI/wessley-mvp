//go:build integration

package natsutil

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func natsURL() string {
	if v := os.Getenv("NATS_URL"); v != "" {
		return v
	}
	return nats.DefaultURL
}

func connectNATS(t *testing.T) *nats.Conn {
	t.Helper()
	nc, err := nats.Connect(natsURL())
	if err != nil {
		t.Fatalf("nats connect: %v", err)
	}
	t.Cleanup(func() { nc.Close() })
	return nc
}

func TestNATS_PubSub(t *testing.T) {
	nc := connectNATS(t)

	type msg struct {
		Text string `json:"text"`
	}

	ch := make(chan msg, 1)
	sub, err := Subscribe(nc, "integ.pubsub", func(ctx context.Context, m msg) {
		ch <- m
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	if err := Publish(context.Background(), nc, "integ.pubsub", msg{Text: "hello integration"}); err != nil {
		t.Fatalf("Publish: %v", err)
	}

	select {
	case got := <-ch:
		if got.Text != "hello integration" {
			t.Fatalf("expected 'hello integration', got %q", got.Text)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestNATS_Request(t *testing.T) {
	nc := connectNATS(t)

	type req struct{ N int }
	type resp struct{ Result int }

	// Responder
	sub, err := nc.Subscribe("integ.request", func(m *nats.Msg) {
		var r req
		if err := json.Unmarshal(m.Data, &r); err != nil {
			return
		}
		data, _ := json.Marshal(resp{Result: r.N * 2})
		m.Respond(data)
	})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	got, err := Request[req, resp](context.Background(), nc, "integ.request", req{N: 21})
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if got.Result != 42 {
		t.Fatalf("expected 42, got %d", got.Result)
	}
}
