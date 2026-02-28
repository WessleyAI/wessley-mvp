package natsutil

import (
	"encoding/json"
	"testing"
)

func TestNatsHeaderCarrierMultipleKeys(t *testing.T) {
	msg := &natsHeaderCarrier{}
	msg.Set("key1", "val1")
	msg.Set("key2", "val2")
	msg.Set("key3", "val3")

	keys := msg.Keys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}

	for _, k := range []string{"key1", "key2", "key3"} {
		found := false
		for _, got := range keys {
			if got == k {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("key %q not found", k)
		}
	}
}

func TestNatsHeaderCarrierOverwrite(t *testing.T) {
	msg := &natsHeaderCarrier{}
	msg.Set("key", "val1")
	msg.Set("key", "val2")
	if got := msg.Get("key"); got != "val2" {
		t.Fatalf("expected val2, got %s", got)
	}
}

func TestNatsHeaderCarrierGetMissing(t *testing.T) {
	msg := &natsHeaderCarrier{}
	msg.Set("exists", "yes")
	if got := msg.Get("nope"); got != "" {
		t.Fatalf("expected empty, got %s", got)
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	type payload struct {
		Items []string `json:"items"`
		Count int      `json:"count"`
	}

	original := payload{Items: []string{"a", "b"}, Count: 2}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}

	var decoded payload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Items) != 2 || decoded.Count != 2 {
		t.Fatalf("roundtrip failed: %+v", decoded)
	}
}

func TestSerializeEmptyStruct(t *testing.T) {
	type empty struct{}
	data, err := json.Marshal(empty{})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "{}" {
		t.Fatalf("expected {}, got %s", data)
	}
}

func TestDeserializeUnknownFields(t *testing.T) {
	data := []byte(`{"name":"test","value":42,"extra":"ignored"}`)
	var msg testMsg
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Name != "test" || msg.Value != 42 {
		t.Fatalf("unexpected: %+v", msg)
	}
}
