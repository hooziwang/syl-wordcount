package textutil

import (
	"strings"
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDetectBinary(t *testing.T) {
	if DetectBinary(nil) {
		t.Fatalf("empty should not be binary")
	}
	if !DetectBinary([]byte{1, 2, 0, 3}) {
		t.Fatalf("nul byte should be binary")
	}
	if !DetectBinary([]byte{1, 2, 3, 4, 5, 6, 7, 'a'}) {
		t.Fatalf("high control-ratio should be binary")
	}
	if DetectBinary([]byte("hello\nworld\t123")) {
		t.Fatalf("plain text should not be binary")
	}
}

func TestDecode(t *testing.T) {
	dec, err := Decode([]byte("hello"))
	if err != nil || dec.Encoding != "utf-8" {
		t.Fatalf("utf8 decode failed: %+v %v", dec, err)
	}

	gbkBytes, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte("中文"))
	if err != nil {
		t.Fatalf("encode gbk: %v", err)
	}
	dec, err = Decode(gbkBytes)
	if err != nil {
		t.Fatalf("gbk decode failed: %v", err)
	}
	if dec.Text != "中文" {
		t.Fatalf("unexpected decoded text: %q", dec.Text)
	}
}

func TestComputeMetricsAndHelpers(t *testing.T) {
	m := ComputeMetrics("\u4f60\u597d\tA\r\nworld\r\n")
	if m.Lines != 2 {
		t.Fatalf("unexpected lines: %d", m.Lines)
	}
	if m.LineEnding != "crlf" {
		t.Fatalf("unexpected line ending: %s", m.LineEnding)
	}
	if m.MaxLineWidth <= 0 || m.AvgLineWidth <= 0 {
		t.Fatalf("unexpected widths: %+v", m)
	}
	if m.Language == "unknown" {
		t.Fatalf("unexpected language: %s", m.Language)
	}

	empty := ComputeMetrics("")
	if empty.Lines != 0 || empty.Chars != 0 || empty.LineEnding != "none" {
		t.Fatalf("unexpected empty metrics: %+v", empty)
	}

	if DisplayWidth("\t") != 4 {
		t.Fatalf("tab width should be 4")
	}
	if DisplayWidth("a\t") != 4 {
		t.Fatalf("tab stop width mismatch")
	}

	if GuessLanguage("hello world") != "en" {
		t.Fatalf("english guess failed")
	}
	if GuessLanguage("中文内容") != "zh" {
		t.Fatalf("zh guess failed")
	}
	if GuessLanguage(" ") != "unknown" {
		t.Fatalf("blank should be unknown")
	}

	if detectLineEnding("a\r\nb\n") != "mixed" {
		t.Fatalf("mixed ending not detected")
	}
}

func TestHashAndPositions(t *testing.T) {
	h := HashSHA256([]byte("abc"))
	if len(h) != 64 {
		t.Fatalf("unexpected sha length: %d", len(h))
	}
	if RuneColumnAtByteOffset("你好A", 0) != 1 {
		t.Fatalf("col at 0 should be 1")
	}
	if RuneColumnAtByteOffset("你好A", 3) != 2 {
		t.Fatalf("col mismatch")
	}

	lines := []string{"abc", "你好A"}
	off := BuildLineOffsets(lines)
	if len(off) != 2 || off[1] <= off[0] {
		t.Fatalf("bad offsets: %#v", off)
	}
	p := LineAndColumnByOffset(lines, off, off[1]+3)
	if p.Line != 2 || p.Column < 2 {
		t.Fatalf("bad pos: %+v", p)
	}

	s := SnippetByRune("0123456789", 5, 2)
	if !strings.Contains(s, "34") {
		t.Fatalf("snippet mismatch: %q", s)
	}
	if SnippetByRune("", 1, 2) != "" {
		t.Fatalf("empty snippet expected")
	}
}

func TestCompilePatternAndClamp(t *testing.T) {
	rx, err := CompilePattern("abc", true)
	if err != nil || !rx.MatchString("abc") {
		t.Fatalf("compile pattern failed: %v", err)
	}
	rx, err = CompilePattern("abc", false)
	if err != nil || !rx.MatchString("ABC") {
		t.Fatalf("case-insensitive compile failed: %v", err)
	}
	if _, err := CompilePattern("(", true); err == nil {
		t.Fatalf("expected invalid regex error")
	}
	if clamp(0, 1, 3) != 1 || clamp(4, 1, 3) != 3 || clamp(2, 1, 3) != 2 {
		t.Fatalf("clamp mismatch")
	}
}
