package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func parseNDJSON(t *testing.T, s string) []map[string]any {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(s), "\n")
	out := make([]map[string]any, 0, len(lines))
	for _, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		m := map[string]any{}
		if err := json.Unmarshal([]byte(ln), &m); err != nil {
			t.Fatalf("invalid json line %q: %v", ln, err)
		}
		out = append(out, m)
	}
	return out
}

func TestVersion(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd(stdout, stderr)
	root.SetArgs([]string{"-v"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(stdout.String(), "syl-wordcount 版本：") {
		t.Fatalf("unexpected output: %q", stdout.String())
	}
}

func TestCheckRequiresConfig(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd(stdout, stderr)
	root.SetArgs([]string{"check", "."})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	ee, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError got %T", err)
	}
	if ee.Code != ExitArg {
		t.Fatalf("unexpected code: %d", ee.Code)
	}
}

func TestStatsOutputNDJSON(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(f, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd(stdout, stderr)
	root.SetArgs([]string{"stats", f})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	events := parseNDJSON(t, stdout.String())
	if len(events) < 3 {
		t.Fatalf("expected >=3 events, got %d", len(events))
	}
	if events[0]["type"] != "meta" {
		t.Fatalf("expected first event meta, got %v", events[0]["type"])
	}
	if events[len(events)-1]["type"] != "summary" {
		t.Fatalf("expected last event summary, got %v", events[len(events)-1]["type"])
	}
}

func TestNormalizeArgs(t *testing.T) {
	got := normalizeArgs([]string{"/tmp/a.txt"})
	if len(got) != 2 || got[0] != "stats" {
		t.Fatalf("unexpected normalize result: %#v", got)
	}
}
