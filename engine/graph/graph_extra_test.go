package graph

import "testing"

func TestStrPropMissing(t *testing.T) {
	props := map[string]any{"name": "test"}
	if got := strProp(props, "missing"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestStrPropNonString(t *testing.T) {
	props := map[string]any{"count": 42}
	if got := strProp(props, "count"); got != "" {
		t.Fatalf("expected empty for non-string, got %q", got)
	}
}

func TestComponentFromPropsEmpty(t *testing.T) {
	c := componentFromProps(map[string]any{})
	if c.ID != "" || c.Name != "" || c.Type != "" {
		t.Fatal("empty props should yield zero-value strings")
	}
	if c.Properties == nil {
		t.Fatal("Properties should be initialized")
	}
}

func TestComponentFromPropsNonStringProp(t *testing.T) {
	props := map[string]any{
		"id":        "c1",
		"prop_num":  42, // non-string value, should be skipped
		"prop_text": "ok",
	}
	c := componentFromProps(props)
	if _, ok := c.Properties["num"]; ok {
		t.Fatal("non-string prop should be skipped")
	}
	if c.Properties["text"] != "ok" {
		t.Fatal("string prop should be included")
	}
}

func TestComponentToMapEmpty(t *testing.T) {
	c := Component{ID: "x"}
	m := componentToMap(c)
	if m["id"] != "x" {
		t.Fatal("missing id")
	}
}

func TestSanitizeRelTypeSpecialChars(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello world!", "HELLOWORLD"},
		{"a.b.c", "ABC"},
		{"123", "123"},
		{"_under_", "_UNDER_"},
		{"!!!@@@", "RELATED_TO"}, // all stripped → fallback
	}
	for _, tt := range tests {
		got := sanitizeRelType(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeRelType(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
