// Package natsutil provides typed NATS publish/subscribe/request helpers
// with OpenTelemetry trace propagation.
package natsutil

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
)

// natsHeaderCarrier adapts nats.Msg headers for OTel TextMapCarrier.
type natsHeaderCarrier nats.Msg

func (c *natsHeaderCarrier) Get(key string) string {
	if c.Header == nil {
		return ""
	}
	return c.Header.Get(key)
}

func (c *natsHeaderCarrier) Set(key, val string) {
	if c.Header == nil {
		c.Header = make(nats.Header)
	}
	c.Header.Set(key, val)
}

func (c *natsHeaderCarrier) Keys() []string {
	if c.Header == nil {
		return nil
	}
	keys := make([]string, 0, len(c.Header))
	for k := range c.Header {
		keys = append(keys, k)
	}
	return keys
}

// Publish serializes v as JSON and publishes to the given subject.
// Trace context from ctx is injected into NATS message headers.
func Publish[T any](ctx context.Context, nc *nats.Conn, subject string, v T) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
	}
	otel.GetTextMapPropagator().Inject(ctx, (*natsHeaderCarrier)(msg))
	return nc.PublishMsg(msg)
}

// Subscribe registers a handler that deserializes JSON messages of type T.
// Trace context is extracted from NATS message headers and passed to the handler.
// Malformed messages are silently dropped.
func Subscribe[T any](nc *nats.Conn, subject string, handler func(context.Context, T)) (*nats.Subscription, error) {
	return nc.Subscribe(subject, func(msg *nats.Msg) {
		var v T
		if err := json.Unmarshal(msg.Data, &v); err != nil {
			return // drop malformed messages
		}
		ctx := otel.GetTextMapPropagator().Extract(context.Background(), (*natsHeaderCarrier)(msg))
		handler(ctx, v)
	})
}

// Request sends a JSON-encoded request and decodes the response.
// Uses nats.DefaultTimeout.
func Request[Req, Resp any](ctx context.Context, nc *nats.Conn, subject string, req Req) (Resp, error) {
	var zero Resp
	data, err := json.Marshal(req)
	if err != nil {
		return zero, err
	}
	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
	}
	otel.GetTextMapPropagator().Inject(ctx, (*natsHeaderCarrier)(msg))
	resp, err := nc.RequestMsg(msg, nats.DefaultTimeout)
	if err != nil {
		return zero, err
	}
	var result Resp
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return zero, err
	}
	return result, nil
}
