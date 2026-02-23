package scan

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateFormat(t *testing.T) {
	if err := ValidateFormat("ndjson"); err != nil {
		t.Fatalf("ndjson should pass: %v", err)
	}
	if err := ValidateFormat("json"); err != nil {
		t.Fatalf("json should pass: %v", err)
	}
	if err := ValidateFormat("xml"); err == nil {
		t.Fatalf("xml should fail")
	}
}

func TestCollectAndIgnore(t *testing.T) {
	tmp := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmp, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "node_modules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".gitignore"), []byte("ignored.txt\nsub/skip/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "ok.txt"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "ignored.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "node_modules", "x.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "sub", "skip"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "sub", "skip", "x.txt"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "sub", "a.log"), []byte("no"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := Collect(Options{
		Paths:          []string{tmp, filepath.Join(tmp, "not-exist")},
		CWD:            tmp,
		IgnorePatterns: []string{"**/*.log"},
	})
	if len(res.Errors) == 0 {
		t.Fatalf("expected missing path error")
	}
	hasOK := false
	for _, f := range res.Files {
		base := filepath.Base(f)
		if base == "ok.txt" {
			hasOK = true
		}
		if base == "ignored.txt" || base == "x.txt" || base == "a.log" {
			t.Fatalf("ignored file should not be present: %#v", res.Files)
		}
	}
	if !hasOK {
		t.Fatalf("ok.txt should be present: %#v", res.Files)
	}
}

func TestCollectSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink on windows may require admin")
	}
	tmp := t.TempDir()
	target := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(target, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	ln := filepath.Join(tmp, "a.link")
	if err := os.Symlink(target, ln); err != nil {
		t.Fatal(err)
	}

	r1 := Collect(Options{Paths: []string{ln}, CWD: tmp, FollowSymlinks: false})
	if len(r1.Files) != 0 {
		t.Fatalf("symlink should be skipped when follow=false")
	}
	if len(r1.Errors) == 0 || r1.Errors[0].Code != "symlink_skipped" {
		t.Fatalf("expected symlink_skipped error: %#v", r1.Errors)
	}

	r2 := Collect(Options{Paths: []string{ln}, CWD: tmp, FollowSymlinks: true})
	if len(r2.Files) != 1 {
		t.Fatalf("symlink should be kept when follow=true: %#v", r2.Files)
	}
}

func TestIsIgnoredRelativePattern(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "a", "b.txt")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	ok := isIgnored(p, false, Options{CWD: tmp, IgnorePatterns: []string{"a/*.txt"}}, nil)
	if !ok {
		t.Fatalf("relative ignore should match")
	}
}
