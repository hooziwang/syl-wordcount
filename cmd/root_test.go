package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

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
	got := stripANSI(stdout.String())
	if !strings.Contains(got, "syl-wordcount 版本：") {
		t.Fatalf("unexpected output: %q", got)
	}
	if !strings.Contains(got, "DADDYLOVESYL") {
		t.Fatalf("missing DADDYLOVESYL banner: %q", got)
	}
}

func TestCheckRequiresRules(t *testing.T) {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "SYL_WC_") {
			t.Skip("当前环境已存在 SYL_WC_* 变量，跳过无规则来源场景测试")
		}
	}
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
	if ee.Code != ExitConfig {
		t.Fatalf("unexpected code: %d", ee.Code)
	}
	if !strings.Contains(ee.Error(), "check 模式需要规则") {
		t.Fatalf("unexpected error message: %v", ee)
	}
}

func TestStatsSubcommandRemoved(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd(stdout, stderr)
	root.SetArgs([]string{"stats", "."})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected unknown command error")
	}
	if !strings.Contains(err.Error(), "unknown command \"stats\"") {
		t.Fatalf("unexpected error: %v", err)
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
	root.SetArgs(normalizeArgs([]string{f}))
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

func TestCheckWithEnvRulesOnly(t *testing.T) {
	t.Setenv("SYL_WC_MAX_CHARS", "1")
	tmp := t.TempDir()
	f := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(f, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd(stdout, stderr)
	root.SetArgs([]string{"check", f})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected violation exit")
	}
	ee, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError got %T", err)
	}
	if ee.Code != ExitViolation {
		t.Fatalf("unexpected code: %d", ee.Code)
	}
	events := parseNDJSON(t, stdout.String())
	hasViolation := false
	for _, e := range events {
		if e["type"] == "violation" {
			hasViolation = true
			break
		}
	}
	if !hasViolation {
		t.Fatalf("expected violation event, got: %s", stdout.String())
	}
	for _, e := range events {
		if e["type"] == "pass" {
			t.Fatalf("default check output should hide pass events: %s", stdout.String())
		}
	}
}

func TestCheckAllIncludesPass(t *testing.T) {
	t.Setenv("SYL_WC_MAX_CHARS", "1")
	tmp := t.TempDir()
	okf := filepath.Join(tmp, "ok.txt")
	badf := filepath.Join(tmp, "bad.txt")
	if err := os.WriteFile(okf, []byte("a"), 0o644); err != nil {
		t.Fatalf("write ok file: %v", err)
	}
	if err := os.WriteFile(badf, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd(stdout, stderr)
	root.SetArgs([]string{"check", okf, badf, "--all"})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected violation exit")
	}
	ee, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected ExitError got %T", err)
	}
	if ee.Code != ExitViolation {
		t.Fatalf("unexpected code: %d", ee.Code)
	}
	events := parseNDJSON(t, stdout.String())
	hasPass := false
	hasViolation := false
	for _, e := range events {
		if e["type"] == "pass" {
			hasPass = true
		}
		if e["type"] == "violation" {
			hasViolation = true
		}
	}
	if !hasPass || !hasViolation {
		t.Fatalf("--all should include pass and violation events, got: %s", stdout.String())
	}
}

func TestNormalizeArgsUsesInternalStats(t *testing.T) {
	got := normalizeArgs([]string{"/tmp/a.txt"})
	if len(got) != 2 || got[0] != "__stats" || got[1] != "/tmp/a.txt" {
		t.Fatalf("unexpected normalize result: %#v", got)
	}
}

func TestFollowSymlinksFlagRemoved(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root := NewRootCmd(stdout, stderr)
	root.SetArgs([]string{"--follow-symlinks", "."})
	err := root.Execute()
	if err == nil {
		t.Fatalf("expected unknown flag error")
	}
	if !strings.Contains(err.Error(), "unknown flag: --follow-symlinks") {
		t.Fatalf("unexpected error: %v", err)
	}
}
