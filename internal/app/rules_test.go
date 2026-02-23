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

func TestEvaluateRulesSectionRules(t *testing.T) {
	text := "# 总览\na\tb\n\n## xxx-章节A\n123456\n### 子节\n123\n\n## yyy-章节B\n123456789012\n"
	t.Run("global_and_section_rules_together", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.md", text), config.Rules{
			NoTabs: true, // 全局规则
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						MaxChars: ip(5),
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		hasFileTab := false
		hasSectionMax := false
		for _, v := range vs {
			if v.RuleID == "no_tabs" && v.Scope == "file" && v.Line == 2 {
				hasFileTab = true
			}
			if v.RuleID == "max_chars" && v.Scope == "section" && v.Line == 4 {
				hasSectionMax = true
			}
		}
		if !hasFileTab || !hasSectionMax {
			t.Fatalf("expected both global and section violations, got: %+v", vs)
		}
	})

	t.Run("multiple_section_rules_with_different_thresholds", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.md", text), config.Rules{
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						MaxChars: ip(5),
					},
				},
				{
					HeadingContains: "yyy",
					Rules: config.SectionScopedRules{
						MaxChars: ip(20),
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if countRule(vs, "max_chars") != 1 {
			t.Fatalf("expected one section max_chars violation, got: %+v", vs)
		}
		v, _ := firstRule(vs, "max_chars")
		if v.Scope != "section" || v.Line != 4 {
			t.Fatalf("unexpected section max_chars violation: %+v", v)
		}
	})

	t.Run("no_match_no_violation", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.md", text), config.Rules{
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "not-exists",
					Rules: config.SectionScopedRules{
						MaxChars: ip(1),
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if hasRule(vs, "max_chars") {
			t.Fatalf("unexpected section violation: %+v", vs)
		}
	})

	t.Run("invalid_section_rule", func(t *testing.T) {
		vs, errs := EvaluateRules(newFC("/tmp/a.md", text), config.Rules{
			SectionRules: []config.SectionRule{
				{HeadingContains: "", Rules: config.SectionScopedRules{MaxChars: ip(10)}},
				{HeadingContains: "xxx"},
			},
		})
		if len(vs) != 0 {
			t.Fatalf("invalid section rules should not produce violations: %+v", vs)
		}
		if len(errs) != 2 {
			t.Fatalf("expected two section rule errors, got %d: %v", len(errs), errs)
		}
	})

	t.Run("line_rules_are_scoped_to_section_content", func(t *testing.T) {
		content := "# 其他\na\tb\n\n## xxx命中\nx\tz\n"
		vs, errs := EvaluateRules(newFC("/tmp/a.md", content), config.Rules{
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						NoTabs: true,
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if countRule(vs, "no_tabs") != 1 {
			t.Fatalf("expected exactly one no_tabs violation in matched section, got: %+v", vs)
		}
		v, _ := firstRule(vs, "no_tabs")
		if v.Scope != "section" || v.Line != 5 {
			t.Fatalf("unexpected scoped line violation: %+v", v)
		}
	})

	t.Run("overlapped_section_rules_accumulate", func(t *testing.T) {
		content := "# xxx\n1234567890\n"
		vs, errs := EvaluateRules(newFC("/tmp/a.md", content), config.Rules{
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						MaxChars: ip(5),
					},
				},
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						MaxLines: ip(0),
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if !hasRule(vs, "max_chars") || !hasRule(vs, "max_lines") {
			t.Fatalf("expected overlapped section rules to accumulate, got: %+v", vs)
		}
	})

	t.Run("global_and_section_same_rule_id_do_not_override", func(t *testing.T) {
		content := "# xxx\n123456\n"
		vs, errs := EvaluateRules(newFC("/tmp/a.md", content), config.Rules{
			MaxChars: ip(5), // 全局触发
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						MaxChars: ip(100), // 章节不触发
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if countRule(vs, "max_chars") != 1 {
			t.Fatalf("expected only one max_chars violation, got: %+v", vs)
		}
		v, _ := firstRule(vs, "max_chars")
		if v.Scope != "file" {
			t.Fatalf("expected global max_chars file-scope violation, got: %+v", v)
		}
	})

	t.Run("section_pattern_compile_error", func(t *testing.T) {
		content := "# xxx\nhello\n"
		vs, errs := EvaluateRules(newFC("/tmp/a.md", content), config.Rules{
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						ForbiddenPatterns: []config.PatternRule{{Pattern: "("}},
					},
				},
			},
		})
		if len(vs) != 0 {
			t.Fatalf("compile error should not produce violations, got: %+v", vs)
		}
		if len(errs) != 1 {
			t.Fatalf("expected one section regex compile error, got %d: %v", len(errs), errs)
		}
	})

	t.Run("nested_heading_boundary", func(t *testing.T) {
		content := "# A\nkeep\n## A child\nchild\n# B\noutside\n"
		vs, errs := EvaluateRules(newFC("/tmp/a.md", content), config.Rules{
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "A",
					Rules: config.SectionScopedRules{
						ForbiddenPatterns: []config.PatternRule{{Pattern: "outside"}},
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		if hasRule(vs, "forbidden_pattern") {
			t.Fatalf("section boundary should exclude next top-level section content, got: %+v", vs)
		}
	})

	t.Run("section_required_pattern_position", func(t *testing.T) {
		content := "# xxx 标题\nabc\n"
		vs, errs := EvaluateRules(newFC("/tmp/a.md", content), config.Rules{
			SectionRules: []config.SectionRule{
				{
					HeadingContains: "xxx",
					Rules: config.SectionScopedRules{
						RequiredPatterns: []config.PatternRule{{Pattern: "MUST", CaseSensitive: bp(true)}},
					},
				},
			},
		})
		if len(errs) != 0 {
			t.Fatalf("unexpected errs: %v", errs)
		}
		v, ok := firstRule(vs, "required_pattern")
		if !ok {
			t.Fatalf("expected required_pattern violation")
		}
		if v.Scope != "section" || v.Line != 1 || v.Column != 1 {
			t.Fatalf("unexpected section required_pattern position: %+v", v)
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

	secs := collectMarkdownSections([]string{"# A", "x", "## B", "y", "# C"})
	if len(secs) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(secs))
	}
	if secs[0].Heading != "A" || secs[0].HeadingLine != 1 {
		t.Fatalf("unexpected first section: %+v", secs[0])
	}
	if normalizeHeadingTitle("Title ### ") != "Title" {
		t.Fatalf("normalizeHeadingTitle failed")
	}
	v2 := scopeLevelViolation("/tmp/a.md", evalScope{
		Scope:     "section",
		Label:     "xxx",
		LabelLine: 7,
	}, "max_chars", "m", 20, 10)
	if v2.Scope != "section" || v2.Line != 7 || v2.Column != 1 {
		t.Fatalf("unexpected scopeLevelViolation: %+v", v2)
	}
}
