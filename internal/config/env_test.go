package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRulesForCheckFromConfig(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "r.yaml")
	if err := os.WriteFile(p, []byte("rules:\n  max_lines: 10\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r, src, err := LoadRulesForCheck(p)
	if err != nil {
		t.Fatalf("load rules from config failed: %v", err)
	}
	if src != p {
		t.Fatalf("unexpected source: %s", src)
	}
	if r.MaxLines == nil || *r.MaxLines != 10 {
		t.Fatalf("unexpected max lines: %#v", r.MaxLines)
	}
}

func TestLoadRulesFromEnv(t *testing.T) {
	t.Setenv("SYL_WC_MAX_LINE_WIDTH", "100")
	t.Setenv("SYL_WC_MAX_CHARS", "5000")
	t.Setenv("SYL_WC_NO_TABS", "true")
	t.Setenv("SYL_WC_ALLOWED_EXTENSIONS", ".md,.txt")
	t.Setenv("SYL_WC_FORBIDDEN_PATTERNS", "TODO,password")
	r, ok, err := LoadRulesFromEnv(EnvPrefix)
	if err != nil {
		t.Fatalf("load from env failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected env rules present")
	}
	if r.MaxLineWidth == nil || *r.MaxLineWidth != 100 {
		t.Fatalf("bad max line width: %#v", r.MaxLineWidth)
	}
	if r.MaxChars == nil || *r.MaxChars != 5000 {
		t.Fatalf("bad max chars: %#v", r.MaxChars)
	}
	if !r.NoTabs {
		t.Fatalf("expected no_tabs=true")
	}
	if len(r.AllowedExtensions) != 2 {
		t.Fatalf("bad extensions: %#v", r.AllowedExtensions)
	}
	if len(r.ForbiddenPatterns) != 2 {
		t.Fatalf("bad forbidden patterns: %#v", r.ForbiddenPatterns)
	}
}

func TestLoadRulesForCheckNoConfigNoEnv(t *testing.T) {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, EnvPrefix) {
			t.Skip("当前环境已存在 SYL_WC_* 变量，跳过无环境变量场景测试")
		}
	}
	r, src, err := LoadRulesForCheck("")
	if err == nil {
		t.Fatalf("expected error, got rules=%+v source=%s", r, src)
	}
}

func TestLoadRulesFromEnvInvalidValue(t *testing.T) {
	t.Setenv("TWC_MAX_CHARS", "abc")
	_, _, err := LoadRulesFromEnv("TWC_")
	if err == nil {
		t.Fatalf("expected invalid int error")
	}
}
