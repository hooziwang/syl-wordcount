package app

import (
	"testing"

	"syl-wordcount/internal/config"
	"syl-wordcount/internal/textutil"
)

func TestEvaluateRulesMaxLineWidth(t *testing.T) {
	text := "123456\n"
	m := textutil.ComputeMetrics(text)
	fc := FileContent{Path: "/tmp/a.txt", Data: []byte(text), Text: text, Metrics: m}
	max := 4
	vs, errs := EvaluateRules(fc, config.Rules{MaxLineWidth: &max})
	if len(errs) != 0 {
		t.Fatalf("unexpected errs: %v", errs)
	}
	if len(vs) == 0 {
		t.Fatalf("expected violation")
	}
	if vs[0].OverflowStartColumn != 5 {
		t.Fatalf("unexpected overflow column: %d", vs[0].OverflowStartColumn)
	}
}

func TestEvaluateRulesRich(t *testing.T) {
	text := "bad\tline  \n\n\n中文　TODO\n"
	m := textutil.ComputeMetrics(text)
	fc := FileContent{Path: "/tmp/a.md", Data: []byte(text), Text: text, Metrics: m}

	minChars := 1
	maxChars := 5
	minLines := 1
	maxLines := 2
	maxLineWidth := 3
	avgLineWidth := 2
	maxBlank := 1
	caseFalse := false
	rules := config.Rules{
		MinChars:                 &minChars,
		MaxChars:                 &maxChars,
		MinLines:                 &minLines,
		MaxLines:                 &maxLines,
		MaxLineWidth:             &maxLineWidth,
		AvgLineWidth:             &avgLineWidth,
		MaxFileSize:              "2B",
		NoTrailingSpaces:         true,
		NoTabs:                   true,
		NoFullwidthSpace:         true,
		MaxConsecutiveBlankLines: &maxBlank,
		AllowedExtensions:        []string{".txt"},
		ForbiddenPatterns: []config.PatternRule{
			{Pattern: "TODO", CaseSensitive: &caseFalse},
			{Pattern: "(", CaseSensitive: &caseFalse}, // invalid regex
		},
		RequiredPatterns: []config.PatternRule{
			{Pattern: "MUST_HAVE", CaseSensitive: &caseFalse},
		},
	}
	vs, errs := EvaluateRules(fc, rules)
	if len(vs) == 0 {
		t.Fatalf("expected rich violations")
	}
	if len(errs) == 0 {
		t.Fatalf("expected regex compile error")
	}
	foundRequired := false
	foundTabs := false
	for _, v := range vs {
		if v.RuleID == "required_pattern" && v.Line == 0 && v.Column == 0 {
			foundRequired = true
		}
		if v.RuleID == "no_tabs" {
			foundTabs = true
		}
	}
	if !foundRequired {
		t.Fatalf("missing required_pattern file-level violation")
	}
	if !foundTabs {
		t.Fatalf("missing no_tabs violation")
	}
}

func TestRuleHelpers(t *testing.T) {
	// cover compileRule, fileLevel, snippetLine
	rx, err := compileRule(config.PatternRule{Pattern: "abc"})
	if err != nil || !rx.MatchString("abc") {
		t.Fatalf("compileRule failed: %v", err)
	}
	v := fileLevel("/tmp/a.txt", "max_chars", "too much", 10, 5)
	if v.Scope != "file" || v.Line != 0 || v.Column != 0 {
		t.Fatalf("unexpected file-level violation: %+v", v)
	}
	s := snippetLine("12345678901234567890123456789012345678901234567890123456789012345678901234567890X")
	if len(s) <= 80 {
		t.Fatalf("snippet should be truncated: %q", s)
	}
}
