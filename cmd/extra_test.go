package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExitErrorString(t *testing.T) {
	if (&ExitError{Code: 2, Msg: "x"}).Error() != "x" {
		t.Fatalf("unexpected exit error msg")
	}
	if (&ExitError{Code: 2}).Error() == "" {
		t.Fatalf("empty code message")
	}
}

func TestParseSize(t *testing.T) {
	v, err := parseSize("10MB")
	if err != nil || v != 10*1024*1024 {
		t.Fatalf("parse mb failed: %d %v", v, err)
	}
	v, err = parseSize("1024")
	if err != nil || v != 1024 {
		t.Fatalf("parse bytes failed: %d %v", v, err)
	}
	if _, err := parseSize("bad"); err == nil {
		t.Fatalf("expected parse error")
	}
}

func TestExecuteExitCodes(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"syl-wordcount"}
	if code := Execute(); code != ExitArg {
		t.Fatalf("expected ExitArg, got %d", code)
	}

	tmp := t.TempDir()
	file := filepath.Join(tmp, "a.txt")
	cfg := filepath.Join(tmp, "rules.yaml")
	if err := os.WriteFile(file, []byte("TODO\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("rules:\n  forbidden_patterns:\n    - pattern: \"TODO\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.Args = []string{"syl-wordcount", "check", file, "--config", cfg}
	if code := Execute(); code != ExitViolation {
		t.Fatalf("expected ExitViolation, got %d", code)
	}

	os.Args = []string{"syl-wordcount", filepath.Join(tmp, "missing.txt")}
	if code := Execute(); code != ExitInput {
		t.Fatalf("expected ExitInput, got %d", code)
	}
}
