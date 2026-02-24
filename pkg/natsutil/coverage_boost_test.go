package natsutil

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
)

func startTestNATS(t *testing.T) (*natsserver.Server, *nats.Conn) {
	t.Helper()
	opts := &natsserver.Options{Port: -1}
	srv, err := natsserver.NewServer(opts)
	if err != nil {
		t.Fatal(err)
	}
	srv.Start()
	if !srv.ReadyForConnections(3 * time.Second) {
		t.Fatal("nats not ready")
	}
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		nc.Close()
		srv.Shutdown()
	})
	return srv, nc
}

type payload struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func TestPublish(t *testing.T) {
	_, nc := startTestNATS(t)

	// Subscribe raw to verify Publish output
	ch := make(chan *nats.Msg, 1)
	sub, err := nc.ChanSubscribe("test.pub", ch)
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	err = Publish(context.Background(), nc, "test.pub", payload{Name: "hello", Value: 1})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case msg := <-ch:
		var p payload
		if err := json.Unmarshal(msg.Data, &p); err != nil {
			t.Fatal(err)
		}
		if p.Name != "hello" || p.Value != 1 {
			t.Fatalf("unexpected payload: %+v", p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestSubscribe(t *testing.T) {
	_, nc := startTestNATS(t)

	ch := make(chan payload, 1)
	sub, err := Subscribe(nc, "test.sub", func(ctx context.Context, p payload) {
		ch <- p
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	// Publish via our helper
	err = Publish(context.Background(), nc, "test.sub", payload{Name: "world", Value: 42})
	if err != nil {
		t.Fatal(err)
	}

	select {
	case p := <-ch:
		if p.Name != "world" || p.Value != 42 {
			t.Fatalf("unexpected: %+v", p)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestSubscribeDropsMalformedReal(t *testing.T) {
	_, nc := startTestNATS(t)

	called := make(chan struct{}, 1)
	sub, err := Subscribe(nc, "test.malformed", func(ctx context.Context, p payload) {
		called <- struct{}{}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	// Send malformed JSON
	nc.Publish("test.malformed", []byte("{bad"))
	nc.Flush()

	select {
	case <-called:
		t.Fatal("handler should not be called for malformed data")
	case <-time.After(100 * time.Millisecond):
		// expected
	}
}

func TestRequest(t *testing.T) {
	_, nc := startTestNATS(t)

	// Set up responder
	sub, err := nc.Subscribe("test.req", func(msg *nats.Msg) {
		var req payload
		json.Unmarshal(msg.Data, &req)
		resp := payload{Name: req.Name + "-resp", Value: req.Value * 2}
		data, _ := json.Marshal(resp)
		msg.Respond(data)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	resp, err := Request[payload, payload](context.Background(), nc, "test.req", payload{Name: "test", Value: 5})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Name != "test-resp" || resp.Value != 10 {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestRequestTimeout(t *testing.T) {
	_, nc := startTestNATS(t)

	// No responder → timeout
	_, err := Request[payload, payload](context.Background(), nc, "test.noreply", payload{Name: "x", Value: 1})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestPublishMarshalError(t *testing.T) {
	_, nc := startTestNATS(t)

	// chan is not JSON-marshalable
	err := Publish(context.Background(), nc, "test.err", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestRequestMarshalError(t *testing.T) {
	_, nc := startTestNATS(t)

	_, err := Request[chan int, payload](context.Background(), nc, "test.err", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestRequestUnmarshalError(t *testing.T) {
	_, nc := startTestNATS(t)

	// Responder sends invalid JSON
	sub, err := nc.Subscribe("test.badjson", func(msg *nats.Msg) {
		msg.Respond([]byte("{invalid"))
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	_, err = Request[payload, payload](context.Background(), nc, "test.badjson", payload{Name: "x", Value: 1})
	if err == nil {
		t.Fatal("expected unmarshal error")
	}
}
