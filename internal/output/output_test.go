package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestWriteNDJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	events := []map[string]any{{"type": "meta"}, {"type": "summary"}}
	if err := Write(buf, "ndjson", events); err != nil {
		t.Fatalf("write ndjson failed: %v", err)
	}
	out := strings.TrimSpace(buf.String())
	lines := strings.Split(out, "\n")
	if len(lines) != 2 {
		t.Fatalf("unexpected lines: %q", out)
	}
}

func TestWriteJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	events := []map[string]any{{"type": "meta"}}
	if err := Write(buf, "json", events); err != nil {
		t.Fatalf("write json failed: %v", err)
	}
	if !strings.Contains(buf.String(), "\"events\"") {
		t.Fatalf("unexpected json: %s", buf.String())
	}
}

func TestWriteInvalidFormat(t *testing.T) {
	if err := Write(&bytes.Buffer{}, "bad", nil); err == nil {
		t.Fatalf("expected format error")
	}
}
