package app

import (
	"testing"

	"syl-wordcount/internal/config"
	"syl-wordcount/internal/textutil"
)

func ip(v int) *int { return &v }

func bp(v bool) *bool { return &v }

func newFC(path, text string) FileContent {
	return FileContent{
		Path:    path,
		Data:    []byte(text),
		Text:    text,
		Metrics: textutil.ComputeMetrics(text),
	}
}

func hasRule(vs []Violation, id string) bool {
	for _, v := range vs {
		if v.RuleID == id {
			return true
		}
	}
	return false
}

func firstRule(vs []Violation, id string) (Violation, bool) {
	for _, v := range vs {
		if v.RuleID == id {
			return v, true
		}
	}
	return Violation{}, false
}

func countRule(vs []Violation, id string) int {
	n := 0
	for _, v := range vs {
		if v.RuleID == id {
			n++
		}
	}
	return n
}

func TestEvaluateRulesFileLevelNumericRules(t *testing.T) {
	t.Run("allowed_extensions", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.md", "ok"), config.Rules{AllowedExtensions: []string{".txt"}})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if !hasRule(vs, "allowed_extensions") {
			t.Fatalf("expected allowed_extensions violation, got: %+v", vs)
		}
		v, _ := firstRule(vs, "allowed_extensions")
		if v.Scope != "file" || v.Line != 0 || v.Column != 0 {
			t.Fatalf("expected file-level violation, got: %+v", v)
		}
	})

	t.Run("allowed_extensions_case_insensitive", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.TxT", "ok"), config.Rules{AllowedExtensions: []string{".txt"}})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if hasRule(vs, "allowed_extensions") {
			t.Fatalf("should pass case-insensitive extension check, got: %+v", vs)
		}
	})

	t.Run("min_chars", func(t *testing.T) {
		vs, _ := EvaluateRules(newFC("/tmp/a.txt", "ab"), config.Rules{MinChars: ip(3)})
		if !hasRule(vs, "min_chars") {
			t.Fatalf("expected min_chars violation, got: %+v", vs)
		}
	})

	t.Run("max_chars", func(t *testing.T) {
		vs, _ := EvaluateRules(newFC("/tmp/a.txt", "abcd"), config.Rules{MaxChars: ip(3)})
		if !hasRule(vs, "max_chars") {
			t.Fatalf("expected max_chars violation, got: %+v", vs)
		}
	})

	t.Run("min_lines", func(t *testing.T) {
		vs, _ := EvaluateRules(newFC("/tmp/a.txt", "one"), config.Rules{MinLines: ip(2)})
		if !hasRule(vs, "min_lines") {
			t.Fatalf("expected min_lines violation, got: %+v", vs)
		}
	})

	t.Run("max_lines", func(t *testing.T) {
		vs, _ := EvaluateRules(newFC("/tmp/a.txt", "a\nb\nc"), config.Rules{MaxLines: ip(2)})
		if !hasRule(vs, "max_lines") {
			t.Fatalf("expected max_lines violation, got: %+v", vs)
		}
	})

	t.Run("avg_line_width", func(t *testing.T) {
		vs, _ := EvaluateRules(newFC("/tmp/a.txt", "abcd\nefgh"), config.Rules{AvgLineWidth: ip(3)})
		if !hasRule(vs, "avg_line_width") {
			t.Fatalf("expected avg_line_width violation, got: %+v", vs)
		}
	})

	t.Run("max_file_size", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "abcd"), config.Rules{MaxFileSize: "3B"})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if !hasRule(vs, "max_file_size") {
			t.Fatalf("expected max_file_size violation, got: %+v", vs)
		}
	})

	t.Run("max_file_size_invalid", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "abcd"), config.Rules{MaxFileSize: "oops"})
		if len(vs) != 0 {
			t.Fatalf("invalid max_file_size should not create violation, got: %+v", vs)
		}
		if len(errs) == 0 {
			t.Fatalf("expected max_file_size parse error")
		}
	})
}

func TestEvaluateRulesLineLevelRules(t *testing.T) {
	t.Run("max_line_width_with_position", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "123456\n"), config.Rules{MaxLineWidth: ip(4)})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		v, ok := firstRule(vs, "max_line_width")
		if !ok {
			t.Fatalf("expected max_line_width violation, got: %+v", vs)
		}
		if v.Line != 1 || v.Column != 5 || v.OverflowStartColumn != 5 || v.LineEndColumn != 6 {
			t.Fatalf("unexpected width position: %+v", v)
		}
	})

	t.Run("no_trailing_spaces", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "ab \n"), config.Rules{NoTrailingSpaces: true})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		v, ok := firstRule(vs, "no_trailing_spaces")
		if !ok {
			t.Fatalf("expected no_trailing_spaces violation, got: %+v", vs)
		}
		if v.Line != 1 || v.Column != 3 {
			t.Fatalf("unexpected trailing-space position: %+v", v)
		}
	})

	t.Run("no_tabs", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "a\tb\n"), config.Rules{NoTabs: true})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		v, ok := firstRule(vs, "no_tabs")
		if !ok {
			t.Fatalf("expected no_tabs violation, got: %+v", vs)
		}
		if v.Line != 1 || v.Column != 2 {
			t.Fatalf("unexpected tab position: %+v", v)
		}
	})

	t.Run("no_fullwidth_space", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "中　文\n"), config.Rules{NoFullwidthSpace: true})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		v, ok := firstRule(vs, "no_fullwidth_space")
		if !ok {
			t.Fatalf("expected no_fullwidth_space violation, got: %+v", vs)
		}
		if v.Line != 1 || v.Column != 2 {
			t.Fatalf("unexpected fullwidth-space position: %+v", v)
		}
	})

	t.Run("max_consecutive_blank_lines", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "a\n\n\nb\n"), config.Rules{MaxConsecutiveBlankLines: ip(1)})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		v, ok := firstRule(vs, "max_consecutive_blank_lines")
		if !ok {
			t.Fatalf("expected max_consecutive_blank_lines violation, got: %+v", vs)
		}
		if v.Line != 3 || v.Column != 1 {
			t.Fatalf("unexpected blank-lines position: %+v", v)
		}
	})
}

func TestEvaluateRulesPatternRules(t *testing.T) {
	t.Run("forbidden_pattern_case_sensitive", func(t *testing.T) {
		vs, errs := EvaluateRules(
			newFC("/tmp/a.txt", "todo\nTODO\n"),
			config.Rules{ForbiddenPatterns: []config.PatternRule{{Pattern: "TODO", CaseSensitive: bp(true)}}},
		)
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if countRule(vs, "forbidden_pattern") != 1 {
			t.Fatalf("expected exactly one case-sensitive forbidden hit, got: %+v", vs)
		}
		v, _ := firstRule(vs, "forbidden_pattern")
		if v.Line != 2 || v.Column != 1 || v.Actual != "TODO" {
			t.Fatalf("unexpected forbidden match position/content: %+v", v)
		}
	})

	t.Run("forbidden_pattern_case_insensitive", func(t *testing.T) {
		vs, errs := EvaluateRules(
			newFC("/tmp/a.txt", "todo\nTODO\n"),
			config.Rules{ForbiddenPatterns: []config.PatternRule{{Pattern: "TODO", CaseSensitive: bp(false)}}},
		)
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if countRule(vs, "forbidden_pattern") != 2 {
			t.Fatalf("expected two case-insensitive forbidden hits, got: %+v", vs)
		}
	})

	t.Run("required_pattern_missing_and_present", func(t *testing.T) {
		miss, errs := EvaluateRules(
			newFC("/tmp/a.txt", "hello\n"),
			config.Rules{RequiredPatterns: []config.PatternRule{{Pattern: "MUST", CaseSensitive: bp(true)}}},
		)
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		v, ok := firstRule(miss, "required_pattern")
		if !ok {
			t.Fatalf("expected required_pattern violation when missing")
		}
		if v.Scope != "file" || v.Line != 0 || v.Column != 0 {
			t.Fatalf("required_pattern should be file-level, got: %+v", v)
		}

		hit, errs := EvaluateRules(
			newFC("/tmp/a.txt", "hello MUST\n"),
			config.Rules{RequiredPatterns: []config.PatternRule{{Pattern: "MUST", CaseSensitive: bp(true)}}},
		)
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if hasRule(hit, "required_pattern") {
			t.Fatalf("required pattern present should not violate, got: %+v", hit)
		}
	})

	t.Run("pattern_compile_errors", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", "hello"), config.Rules{
			ForbiddenPatterns: []config.PatternRule{{Pattern: "("}},
			RequiredPatterns:  []config.PatternRule{{Pattern: "("}},
		})
		if len(vs) != 0 {
			t.Fatalf("invalid regex should not produce violation, got: %+v", vs)
		}
		if len(errs) != 2 {
			t.Fatalf("expected two compile errors, got %d: %v", len(errs), errs)
		}
	})
}

func TestEvaluateRulesCombinations(t *testing.T) {
	t.Run("style_combo", func(t *testing.T) {
		text := "abcd\t  \n\n\n"
		vs, errs := EvaluateRules(newFC("/tmp/a.txt", text), config.Rules{
			MaxLineWidth:             ip(4),
			NoTrailingSpaces:         true,
			NoTabs:                   true,
			MaxConsecutiveBlankLines: ip(1),
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		want := []string{
			"max_line_width",
			"no_trailing_spaces",
			"no_tabs",
			"max_consecutive_blank_lines",
		}
		for _, id := range want {
			if !hasRule(vs, id) {
				t.Fatalf("missing %s in combo result: %+v", id, vs)
			}
		}
	})

	t.Run("mixed_file_level_combo", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.md", "abc"), config.Rules{
			AllowedExtensions: []string{".txt"},
			MinChars:          ip(5),
			MaxChars:          ip(2),
			MinLines:          ip(2),
			MaxFileSize:       "2B",
			RequiredPatterns:  []config.PatternRule{{Pattern: "MUST", CaseSensitive: bp(true)}},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		want := []string{
			"allowed_extensions",
			"min_chars",
			"max_chars",
			"min_lines",
			"max_file_size",
			"required_pattern",
		}
		for _, id := range want {
			if !hasRule(vs, id) {
				t.Fatalf("missing %s in combo result: %+v", id, vs)
			}
		}
	})
}

func TestRuleHelpers(t *testing.T) {
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
