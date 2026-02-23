package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func findEvent(events []map[string]any, typ string) map[string]any {
	for _, e := range events {
		if e["type"] == typ {
			return e
		}
	}
	return nil
}

func countEvent(events []map[string]any, typ string) int {
	n := 0
	for _, e := range events {
		if e["type"] == typ {
			n++
		}
	}
	return n
}

func TestDefaultJobs(t *testing.T) {
	n := DefaultJobs()
	if n < 1 || n > 8 {
		t.Fatalf("unexpected jobs: %d", n)
	}
}

func TestRunStatsIncludesHash(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(f, []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Options{
		Mode:    ModeStats,
		Paths:   []string{f},
		CWD:     tmp,
		Format:  "ndjson",
		Version: "test",
	})
	if err != nil {
		t.Fatalf("run stats failed: %v", err)
	}
	if findEvent(res.Events, "meta") == nil {
		t.Fatalf("missing meta event")
	}
	fs := findEvent(res.Events, "file_stats")
	if fs == nil {
		t.Fatalf("missing file_stats")
	}
	if _, ok := fs["hash"]; !ok {
		t.Fatalf("expected hash field")
	}
	sm := findEvent(res.Events, "summary")
	if sm == nil || sm["exit_code"].(int) != 0 {
		t.Fatalf("unexpected summary: %#v", sm)
	}
}

func TestRunCheckPassAndViolation(t *testing.T) {
	tmp := t.TempDir()
	okf := filepath.Join(tmp, "ok.txt")
	badf := filepath.Join(tmp, "bad.txt")
	if err := os.WriteFile(okf, []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(badf, []byte("TODO\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(tmp, "rules.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  forbidden_patterns:\n    - pattern: \"TODO\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Options{
		Mode:       ModeCheck,
		Paths:      []string{okf, badf},
		CWD:        tmp,
		Format:     "ndjson",
		ConfigPath: cfg,
		Version:    "test",
	})
	if err != nil {
		t.Fatalf("run check failed: %v", err)
	}
	if !res.HasViolation {
		t.Fatalf("expected violation")
	}
	if countEvent(res.Events, "pass") == 0 {
		t.Fatalf("expected at least one pass event")
	}
	if countEvent(res.Events, "violation") == 0 {
		t.Fatalf("expected violation event")
	}
	sm := findEvent(res.Events, "summary")
	if sm == nil {
		t.Fatalf("missing summary")
	}
	if sm["violation_count"].(int) < 1 {
		t.Fatalf("summary violation count mismatch: %#v", sm)
	}
}

func TestRunInputErrorsAndSkips(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing.txt")
	big := filepath.Join(tmp, "big.txt")
	if err := os.WriteFile(big, []byte(strings.Repeat("a", 200)), 0o644); err != nil {
		t.Fatal(err)
	}
	binf := filepath.Join(tmp, "bin.dat")
	if err := os.WriteFile(binf, []byte{0, 1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Run(Options{
		Mode:             ModeStats,
		Paths:            []string{missing, big, binf},
		CWD:              tmp,
		Format:           "ndjson",
		MaxFileSizeBytes: 10,
		Version:          "test",
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	if !res.HasInputErr {
		t.Fatalf("expected input error flag")
	}
	if countEvent(res.Events, "error") == 0 {
		t.Fatalf("expected error events")
	}
	for _, e := range res.Events {
		if e["type"] != "error" {
			continue
		}
		if _, ok := e["next_action"]; !ok {
			t.Fatalf("error event missing next_action: %#v", e)
		}
		if _, ok := e["fix_example"]; !ok {
			t.Fatalf("error event missing fix_example: %#v", e)
		}
		if _, ok := e["doc_key"]; !ok {
			t.Fatalf("error event missing doc_key: %#v", e)
		}
		if _, ok := e["recoverable"]; !ok {
			t.Fatalf("error event missing recoverable: %#v", e)
		}
	}
	sm := findEvent(res.Events, "summary")
	if sm == nil {
		t.Fatalf("missing summary")
	}
	if sm["exit_code"].(int) != 3 {
		t.Fatalf("expected summary exit_code 3, got %#v", sm)
	}
	if sm["error_count"].(int) < 1 {
		t.Fatalf("expected error_count >= 1, got %#v", sm)
	}
}

func TestRunNoFilesButHasInputErrorSummary(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "missing.txt")
	res, err := Run(Options{
		Mode:    ModeStats,
		Paths:   []string{missing},
		CWD:     tmp,
		Format:  "ndjson",
		Version: "test",
	})
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	sm := findEvent(res.Events, "summary")
	if sm == nil {
		t.Fatalf("missing summary")
	}
	if sm["total_files"].(int) != 0 {
		t.Fatalf("expected total_files=0, got %#v", sm)
	}
	if sm["exit_code"].(int) != 3 {
		t.Fatalf("expected summary exit_code 3, got %#v", sm)
	}
	if sm["error_count"].(int) < 1 {
		t.Fatalf("expected error_count >= 1, got %#v", sm)
	}
}

func TestRunConfigErrors(t *testing.T) {
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "SYL_WC_") {
			t.Skip("当前环境已存在 SYL_WC_* 变量，跳过无规则来源场景测试")
		}
	}
	tmp := t.TempDir()
	_, err := Run(Options{Mode: ModeCheck, Paths: []string{tmp}, CWD: tmp})
	if err == nil {
		t.Fatalf("expected config error")
	}
	if _, ok := err.(*ConfigErr); !ok {
		t.Fatalf("expected ConfigErr, got %T", err)
	}

	cfg := filepath.Join(tmp, "bad.yaml")
	if err := os.WriteFile(cfg, []byte("rules:\n  unknown_field: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Run(Options{Mode: ModeCheck, Paths: []string{tmp}, CWD: tmp, ConfigPath: cfg})
	if err == nil {
		t.Fatalf("expected config load error")
	}
}

func TestHelpers(t *testing.T) {
	sm := buildSummary(ModeStats, Summary{Processed: 1}, 3)
	if sm["type"] != "summary" || sm["exit_code"].(int) != 3 {
		t.Fatalf("bad summary: %#v", sm)
	}
	if (&ConfigErr{Msg: "a"}).Error() != "a" {
		t.Fatalf("config err string mismatch")
	}
	if (&ArgErr{Msg: "b"}).Error() != "b" {
		t.Fatalf("arg err string mismatch")
	}
	if decideExitCode(Result{HasInternalErr: true}) != 5 {
		t.Fatalf("internal code mismatch")
	}
	if decideExitCode(Result{HasConfigErr: true}) != 4 {
		t.Fatalf("config code mismatch")
	}
	if decideExitCode(Result{HasInputErr: true}) != 3 {
		t.Fatalf("input code mismatch")
	}
	if decideExitCode(Result{HasViolation: true}) != 1 {
		t.Fatalf("violation code mismatch")
	}
	if decideExitCode(Result{}) != 0 {
		t.Fatalf("ok code mismatch")
	}
}

func TestNormalizePaths(t *testing.T) {
	tmp := t.TempDir()
	a := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(a, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := NormalizePaths([]string{"", "a.txt", a, "./a.txt"}, tmp)
	if len(got) != 1 || got[0] != a {
		t.Fatalf("normalize mismatch: %#v", got)
	}
}
