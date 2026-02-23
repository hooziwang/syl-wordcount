package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"syl-wordcount/internal/config"
	"syl-wordcount/internal/textutil"
)

const contextChars = 40

var mdHeadingRegex = regexp.MustCompile(`^\s{0,3}(#{1,6})\s+(.+?)\s*$`)

type FileContent struct {
	Path     string
	Data     []byte
	Text     string
	Encoding string
	Metrics  textutil.Metrics
}

type Violation struct {
	RuleID              string
	Message             string
	Path                string
	Line                int
	Column              int
	OverflowStartColumn int
	LineEndColumn       int
	Snippet             string
	Actual              any
	Limit               any
	Scope               string
}

type evalScope struct {
	Scope     string
	Label     string
	LabelLine int
	Text      string
	Metrics   textutil.Metrics
	StartLine int
}

type markdownSection struct {
	Heading     string
	HeadingLine int
	Level       int
	StartLine   int
	EndLine     int
	Text        string
	Metrics     textutil.Metrics
}

type headingPos struct {
	Title string
	Line  int
	Level int
}

type compiledPattern struct {
	Source string
	Regex  *regexp.Regexp
}

type scopeRules struct {
	MinChars                 *int
	MaxChars                 *int
	MinLines                 *int
	MaxLines                 *int
	MaxLineWidth             *int
	AvgLineWidth             *int
	NoTrailingSpaces         bool
	NoTabs                   bool
	NoFullwidthSpace         bool
	MaxConsecutiveBlankLines *int
	ForbiddenPatterns        []config.PatternRule
	RequiredPatterns         []config.PatternRule
}

type compiledScopeRules struct {
	Rules     scopeRules
	Forbidden []compiledPattern
	Required  []compiledPattern
}

func EvaluateRules(fc FileContent, rules config.Rules) ([]Violation, []error) {
	violations := make([]Violation, 0)
	errs := make([]error, 0)

	if len(rules.AllowedExtensions) > 0 {
		ext := strings.ToLower(filepath.Ext(fc.Path))
		ok := false
		for _, a := range rules.AllowedExtensions {
			if strings.ToLower(strings.TrimSpace(a)) == ext {
				ok = true
				break
			}
		}
		if !ok {
			violations = append(violations, fileLevel(fc.Path, "allowed_extensions", "文件扩展名不在允许范围", ext, rules.AllowedExtensions))
		}
	}

	if s := strings.TrimSpace(rules.MaxFileSize); s != "" {
		maxBytes, err := config.ParseSizeToBytes(s)
		if err != nil {
			errs = append(errs, fmt.Errorf("max_file_size 配置错误：%w", err))
		} else if maxBytes > 0 && int64(len(fc.Data)) > maxBytes {
			violations = append(violations, fileLevel(fc.Path, "max_file_size", "文件大小超出上限", len(fc.Data), maxBytes))
		}
	}

	globalSR := scopeRulesFromGlobal(rules)
	if hasAnyScopeRule(globalSR) {
		compiled, cErrs := compileScopeRules(globalSR, "")
		errs = append(errs, cErrs...)
		fileScope := evalScope{
			Scope:     "file",
			Text:      fc.Text,
			Metrics:   fc.Metrics,
			StartLine: 1,
		}
		violations = append(violations, evaluateScope(fc.Path, fileScope, compiled)...)
	}

	sections := collectMarkdownSections(fc.Metrics.LinesText)
	for i, sr := range rules.SectionRules {
		heading := strings.TrimSpace(sr.HeadingContains)
		if heading == "" {
			errs = append(errs, fmt.Errorf("section_rules[%d] 缺少 heading_contains", i))
			continue
		}

		srScope := scopeRulesFromSection(sr.Rules)
		if !hasAnyScopeRule(srScope) {
			errs = append(errs, fmt.Errorf("section_rules[%d].rules 至少要设置一条规则", i))
			continue
		}

		compiled, cErrs := compileScopeRules(srScope, fmt.Sprintf("section_rules[%d].rules.", i))
		errs = append(errs, cErrs...)

		for _, sec := range sections {
			if !strings.Contains(sec.Heading, heading) {
				continue
			}
			scope := evalScope{
				Scope:     "section",
				Label:     sec.Heading,
				LabelLine: sec.HeadingLine,
				Text:      sec.Text,
				Metrics:   sec.Metrics,
				StartLine: sec.StartLine,
			}
			violations = append(violations, evaluateScope(fc.Path, scope, compiled)...)
		}
	}

	return violations, errs
}

func scopeRulesFromGlobal(r config.Rules) scopeRules {
	return scopeRules{
		MinChars:                 r.MinChars,
		MaxChars:                 r.MaxChars,
		MinLines:                 r.MinLines,
		MaxLines:                 r.MaxLines,
		MaxLineWidth:             r.MaxLineWidth,
		AvgLineWidth:             r.AvgLineWidth,
		NoTrailingSpaces:         r.NoTrailingSpaces,
		NoTabs:                   r.NoTabs,
		NoFullwidthSpace:         r.NoFullwidthSpace,
		MaxConsecutiveBlankLines: r.MaxConsecutiveBlankLines,
		ForbiddenPatterns:        r.ForbiddenPatterns,
		RequiredPatterns:         r.RequiredPatterns,
	}
}

func scopeRulesFromSection(r config.SectionScopedRules) scopeRules {
	return scopeRules{
		MinChars:                 r.MinChars,
		MaxChars:                 r.MaxChars,
		MinLines:                 r.MinLines,
		MaxLines:                 r.MaxLines,
		MaxLineWidth:             r.MaxLineWidth,
		AvgLineWidth:             r.AvgLineWidth,
		NoTrailingSpaces:         r.NoTrailingSpaces,
		NoTabs:                   r.NoTabs,
		NoFullwidthSpace:         r.NoFullwidthSpace,
		MaxConsecutiveBlankLines: r.MaxConsecutiveBlankLines,
		ForbiddenPatterns:        r.ForbiddenPatterns,
		RequiredPatterns:         r.RequiredPatterns,
	}
}

func hasAnyScopeRule(r scopeRules) bool {
	if r.MinChars != nil || r.MaxChars != nil || r.MinLines != nil || r.MaxLines != nil || r.MaxLineWidth != nil || r.AvgLineWidth != nil || r.MaxConsecutiveBlankLines != nil {
		return true
	}
	if r.NoTrailingSpaces || r.NoTabs || r.NoFullwidthSpace {
		return true
	}
	if len(r.ForbiddenPatterns) > 0 || len(r.RequiredPatterns) > 0 {
		return true
	}
	return false
}

func compileScopeRules(r scopeRules, prefix string) (compiledScopeRules, []error) {
	forbidden, ferrs := compilePatternRules(r.ForbiddenPatterns, prefix+"forbidden_patterns")
	required, rerrs := compilePatternRules(r.RequiredPatterns, prefix+"required_patterns")
	errs := make([]error, 0, len(ferrs)+len(rerrs))
	errs = append(errs, ferrs...)
	errs = append(errs, rerrs...)
	return compiledScopeRules{Rules: r, Forbidden: forbidden, Required: required}, errs
}

func evaluateScope(path string, scope evalScope, cr compiledScopeRules) []Violation {
	violations := make([]Violation, 0)
	violations = append(violations, evaluateScalarRules(path, scope, cr.Rules)...)
	violations = append(violations, evaluateLineRules(path, scope, cr.Rules)...)
	violations = append(violations, evaluateForbiddenPatterns(path, scope, cr.Forbidden)...)
	violations = append(violations, evaluateRequiredPatterns(path, scope, cr.Required)...)
	return violations
}

func evaluateScalarRules(path string, scope evalScope, rules scopeRules) []Violation {
	violations := make([]Violation, 0)

	if rules.MinChars != nil && scope.Metrics.Chars < *rules.MinChars {
		violations = append(violations, scopeLevelViolation(path, scope, "min_chars", scopeMessage(scope, "字符数低于下限"), scope.Metrics.Chars, *rules.MinChars))
	}
	if rules.MaxChars != nil && scope.Metrics.Chars > *rules.MaxChars {
		violations = append(violations, scopeLevelViolation(path, scope, "max_chars", scopeMessage(scope, "字符数超出上限"), scope.Metrics.Chars, *rules.MaxChars))
	}
	if rules.MinLines != nil && scope.Metrics.Lines < *rules.MinLines {
		violations = append(violations, scopeLevelViolation(path, scope, "min_lines", scopeMessage(scope, "行数低于下限"), scope.Metrics.Lines, *rules.MinLines))
	}
	if rules.MaxLines != nil && scope.Metrics.Lines > *rules.MaxLines {
		violations = append(violations, scopeLevelViolation(path, scope, "max_lines", scopeMessage(scope, "行数超出上限"), scope.Metrics.Lines, *rules.MaxLines))
	}
	if rules.AvgLineWidth != nil && scope.Metrics.AvgLineWidth > *rules.AvgLineWidth {
		violations = append(violations, scopeLevelViolation(path, scope, "avg_line_width", scopeMessage(scope, "平均行宽超出上限"), scope.Metrics.AvgLineWidth, *rules.AvgLineWidth))
	}

	return violations
}

func evaluateLineRules(path string, scope evalScope, rules scopeRules) []Violation {
	violations := make([]Violation, 0)

	if rules.MaxLineWidth != nil {
		for i, ln := range scope.Metrics.LinesText {
			w := textutil.DisplayWidth(ln)
			if w <= *rules.MaxLineWidth {
				continue
			}
			violations = append(violations, Violation{
				RuleID:              "max_line_width",
				Message:             scopeMessage(scope, "行宽超出上限"),
				Path:                path,
				Line:                scope.StartLine + i,
				Column:              *rules.MaxLineWidth + 1,
				OverflowStartColumn: *rules.MaxLineWidth + 1,
				LineEndColumn:       w,
				Snippet:             snippetLine(ln),
				Actual:              w,
				Limit:               *rules.MaxLineWidth,
				Scope:               scope.Scope,
			})
		}
	}

	if rules.NoTrailingSpaces {
		for i, ln := range scope.Metrics.LinesText {
			j := len(ln)
			for j > 0 {
				r, size := utf8.DecodeLastRuneInString(ln[:j])
				if r != ' ' && r != '\t' {
					break
				}
				j -= size
			}
			if j == len(ln) {
				continue
			}
			col := utf8.RuneCountInString(ln[:j]) + 1
			violations = append(violations, Violation{
				RuleID:  "no_trailing_spaces",
				Message: scopeMessage(scope, "存在行尾空白"),
				Path:    path,
				Line:    scope.StartLine + i,
				Column:  col,
				Snippet: snippetLine(ln),
				Actual:  "trailing_spaces",
				Limit:   "none",
				Scope:   scope.Scope,
			})
		}
	}

	if rules.NoTabs {
		for i, ln := range scope.Metrics.LinesText {
			for b := 0; b < len(ln); {
				r, size := utf8.DecodeRuneInString(ln[b:])
				if r == '\t' {
					violations = append(violations, Violation{
						RuleID:  "no_tabs",
						Message: scopeMessage(scope, "存在制表符"),
						Path:    path,
						Line:    scope.StartLine + i,
						Column:  utf8.RuneCountInString(ln[:b]) + 1,
						Snippet: snippetLine(ln),
						Actual:  "\\t",
						Limit:   "none",
						Scope:   scope.Scope,
					})
				}
				b += size
			}
		}
	}

	if rules.NoFullwidthSpace {
		for i, ln := range scope.Metrics.LinesText {
			for b := 0; b < len(ln); {
				r, size := utf8.DecodeRuneInString(ln[b:])
				if r == '　' {
					violations = append(violations, Violation{
						RuleID:  "no_fullwidth_space",
						Message: scopeMessage(scope, "存在全角空格"),
						Path:    path,
						Line:    scope.StartLine + i,
						Column:  utf8.RuneCountInString(ln[:b]) + 1,
						Snippet: snippetLine(ln),
						Actual:  "U+3000",
						Limit:   "none",
						Scope:   scope.Scope,
					})
				}
				b += size
			}
		}
	}

	if rules.MaxConsecutiveBlankLines != nil {
		blank := 0
		for i, ln := range scope.Metrics.LinesText {
			if strings.TrimSpace(ln) == "" {
				blank++
			} else {
				blank = 0
			}
			if blank > *rules.MaxConsecutiveBlankLines {
				violations = append(violations, Violation{
					RuleID:  "max_consecutive_blank_lines",
					Message: scopeMessage(scope, "连续空行超出上限"),
					Path:    path,
					Line:    scope.StartLine + i,
					Column:  1,
					Snippet: snippetLine(ln),
					Actual:  blank,
					Limit:   *rules.MaxConsecutiveBlankLines,
					Scope:   scope.Scope,
				})
			}
		}
	}

	return violations
}

func evaluateForbiddenPatterns(path string, scope evalScope, rules []compiledPattern) []Violation {
	if len(rules) == 0 {
		return nil
	}
	normText := normalize(scope.Text)
	lineOffsets := textutil.BuildLineOffsets(scope.Metrics.LinesText)
	violations := make([]Violation, 0)
	for _, pr := range rules {
		idxs := pr.Regex.FindAllStringIndex(normText, -1)
		for _, idx := range idxs {
			pos := textutil.LineAndColumnByOffset(scope.Metrics.LinesText, lineOffsets, idx[0])
			line := 0
			if pos.Line > 0 {
				line = scope.StartLine + pos.Line - 1
			}
			violations = append(violations, Violation{
				RuleID:  "forbidden_pattern",
				Message: scopeMessage(scope, "命中禁止模式"),
				Path:    path,
				Line:    line,
				Column:  pos.Column,
				Snippet: textutil.SnippetByRune(normText, idx[0], contextChars),
				Actual:  normText[idx[0]:idx[1]],
				Limit:   pr.Source,
				Scope:   scope.Scope,
			})
		}
	}
	return violations
}

func evaluateRequiredPatterns(path string, scope evalScope, rules []compiledPattern) []Violation {
	if len(rules) == 0 {
		return nil
	}
	normText := normalize(scope.Text)
	violations := make([]Violation, 0)
	for _, pr := range rules {
		if pr.Regex.MatchString(normText) {
			continue
		}
		violations = append(violations, scopeLevelViolation(path, scope, "required_pattern", scopeMessage(scope, "缺少必需模式"), "not_found", pr.Source))
	}
	return violations
}

func compilePatternRules(list []config.PatternRule, kind string) ([]compiledPattern, []error) {
	out := make([]compiledPattern, 0, len(list))
	errs := make([]error, 0)
	for _, pr := range list {
		if strings.TrimSpace(pr.Pattern) == "" {
			continue
		}
		rx, err := compileRule(pr)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s 编译失败：%w", kind, err))
			continue
		}
		out = append(out, compiledPattern{Source: pr.Pattern, Regex: rx})
	}
	return out, errs
}

func collectMarkdownSections(lines []string) []markdownSection {
	headings := make([]headingPos, 0)
	for i, ln := range lines {
		m := mdHeadingRegex.FindStringSubmatch(ln)
		if len(m) != 3 {
			continue
		}
		title := normalizeHeadingTitle(m[2])
		if title == "" {
			continue
		}
		headings = append(headings, headingPos{Title: title, Line: i, Level: len(m[1])})
	}
	if len(headings) == 0 {
		return nil
	}

	sections := make([]markdownSection, 0, len(headings))
	for i, h := range headings {
		end := len(lines) - 1
		for j := i + 1; j < len(headings); j++ {
			if headings[j].Level <= h.Level {
				end = headings[j].Line - 1
				break
			}
		}
		contentStart := h.Line + 1
		content := ""
		if contentStart <= end {
			content = strings.Join(lines[contentStart:end+1], "\n")
		}
		sections = append(sections, markdownSection{
			Heading:     h.Title,
			HeadingLine: h.Line + 1,
			Level:       h.Level,
			StartLine:   contentStart + 1,
			EndLine:     end + 1,
			Text:        content,
			Metrics:     textutil.ComputeMetrics(content),
		})
	}
	return sections
}

func normalizeHeadingTitle(s string) string {
	title := strings.TrimSpace(s)
	title = strings.TrimSpace(strings.TrimRight(title, "#"))
	return title
}

func scopeMessage(scope evalScope, msg string) string {
	if scope.Scope == "section" {
		return fmt.Sprintf("章节[%s]%s", scope.Label, msg)
	}
	return msg
}

func scopeLevelViolation(path string, scope evalScope, ruleID, msg string, actual, limit any) Violation {
	if scope.Scope == "section" {
		return Violation{
			RuleID:  ruleID,
			Message: msg,
			Path:    path,
			Line:    scope.LabelLine,
			Column:  1,
			Snippet: snippetLine(scope.Label),
			Actual:  actual,
			Limit:   limit,
			Scope:   "section",
		}
	}
	return fileLevel(path, ruleID, msg, actual, limit)
}

func compileRule(pr config.PatternRule) (*regexp.Regexp, error) {
	caseSensitive := true
	if pr.CaseSensitive != nil {
		caseSensitive = *pr.CaseSensitive
	}
	return textutil.CompilePattern(pr.Pattern, caseSensitive)
}

func fileLevel(path, ruleID, msg string, actual, limit any) Violation {
	return Violation{
		RuleID:  ruleID,
		Message: msg,
		Path:    path,
		Line:    0,
		Column:  0,
		Snippet: "",
		Actual:  actual,
		Limit:   limit,
		Scope:   "file",
	}
}

func snippetLine(ln string) string {
	r := []rune(ln)
	if len(r) <= 80 {
		return ln
	}
	return string(r[:80]) + "..."
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return s
}
