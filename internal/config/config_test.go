package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("MAX_LINES", "99")
	src := "rules:\n  max_lines: ${MAX_LINES}\n  max_chars: ${MAX_CHARS:-123}\n"
	got, err := expandEnv(src)
	if err != nil {
		t.Fatalf("expand env failed: %v", err)
	}
	if got == src {
		t.Fatalf("expected expanded output")
	}
}

func TestLoadKnownFields(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "c.yaml")
	if err := os.WriteFile(p, []byte("rules:\n  max_lines: 10\n  section_rules:\n    - heading_contains: \"xxx\"\n      rules:\n        max_chars: 200\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if cfg.Rules.MaxLines == nil || *cfg.Rules.MaxLines != 10 {
		t.Fatalf("unexpected max lines: %#v", cfg.Rules.MaxLines)
	}
	if len(cfg.Rules.SectionRules) != 1 {
		t.Fatalf("unexpected section rules: %#v", cfg.Rules.SectionRules)
	}
	if cfg.Rules.SectionRules[0].HeadingContains != "xxx" {
		t.Fatalf("unexpected heading_contains: %#v", cfg.Rules.SectionRules[0])
	}
	if cfg.Rules.SectionRules[0].Rules.MaxChars == nil || *cfg.Rules.SectionRules[0].Rules.MaxChars != 200 {
		t.Fatalf("unexpected section rules max_chars: %#v", cfg.Rules.SectionRules[0].Rules.MaxChars)
	}
}
