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

	if rules.MinChars != nil && fc.Metrics.Chars < *rules.MinChars {
		violations = append(violations, fileLevel(fc.Path, "min_chars", "字符数低于下限", fc.Metrics.Chars, *rules.MinChars))
	}
	if rules.MaxChars != nil && fc.Metrics.Chars > *rules.MaxChars {
		violations = append(violations, fileLevel(fc.Path, "max_chars", "字符数超出上限", fc.Metrics.Chars, *rules.MaxChars))
	}
	if rules.MinLines != nil && fc.Metrics.Lines < *rules.MinLines {
		violations = append(violations, fileLevel(fc.Path, "min_lines", "行数低于下限", fc.Metrics.Lines, *rules.MinLines))
	}
	if rules.MaxLines != nil && fc.Metrics.Lines > *rules.MaxLines {
		violations = append(violations, fileLevel(fc.Path, "max_lines", "行数超出上限", fc.Metrics.Lines, *rules.MaxLines))
	}
	if rules.AvgLineWidth != nil && fc.Metrics.AvgLineWidth > *rules.AvgLineWidth {
		violations = append(violations, fileLevel(fc.Path, "avg_line_width", "平均行宽超出上限", fc.Metrics.AvgLineWidth, *rules.AvgLineWidth))
	}

	if s := strings.TrimSpace(rules.MaxFileSize); s != "" {
		maxBytes, err := config.ParseSizeToBytes(s)
		if err != nil {
			errs = append(errs, fmt.Errorf("max_file_size 配置错误：%w", err))
		} else if maxBytes > 0 && int64(len(fc.Data)) > maxBytes {
			violations = append(violations, fileLevel(fc.Path, "max_file_size", "文件大小超出上限", len(fc.Data), maxBytes))
		}
	}

	lineOffsets := textutil.BuildLineOffsets(fc.Metrics.LinesText)
	normText := normalize(fc.Text)

	if rules.MaxLineWidth != nil {
		for i, ln := range fc.Metrics.LinesText {
			w := textutil.DisplayWidth(ln)
			if w <= *rules.MaxLineWidth {
				continue
			}
			violations = append(violations, Violation{
				RuleID:              "max_line_width",
				Message:             "行宽超出上限",
				Path:                fc.Path,
				Line:                i + 1,
				Column:              *rules.MaxLineWidth + 1,
				OverflowStartColumn: *rules.MaxLineWidth + 1,
				LineEndColumn:       w,
				Snippet:             snippetLine(ln),
				Actual:              w,
				Limit:               *rules.MaxLineWidth,
				Scope:               "line",
			})
		}
	}

	if rules.NoTrailingSpaces {
		for i, ln := range fc.Metrics.LinesText {
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
				Message: "存在行尾空白",
				Path:    fc.Path,
				Line:    i + 1,
				Column:  col,
				Snippet: snippetLine(ln),
				Actual:  "trailing_spaces",
				Limit:   "none",
				Scope:   "line",
			})
		}
	}

	if rules.NoTabs {
		for i, ln := range fc.Metrics.LinesText {
			for b := 0; b < len(ln); {
				r, size := utf8.DecodeRuneInString(ln[b:])
				if r == '\t' {
					violations = append(violations, Violation{
						RuleID:  "no_tabs",
						Message: "存在制表符",
						Path:    fc.Path,
						Line:    i + 1,
						Column:  utf8.RuneCountInString(ln[:b]) + 1,
						Snippet: snippetLine(ln),
						Actual:  "\\t",
						Limit:   "none",
						Scope:   "line",
					})
				}
				b += size
			}
		}
	}

	if rules.NoFullwidthSpace {
		for i, ln := range fc.Metrics.LinesText {
			for b := 0; b < len(ln); {
				r, size := utf8.DecodeRuneInString(ln[b:])
				if r == '　' {
					violations = append(violations, Violation{
						RuleID:  "no_fullwidth_space",
						Message: "存在全角空格",
						Path:    fc.Path,
						Line:    i + 1,
						Column:  utf8.RuneCountInString(ln[:b]) + 1,
						Snippet: snippetLine(ln),
						Actual:  "U+3000",
						Limit:   "none",
						Scope:   "line",
					})
				}
				b += size
			}
		}
	}

	if rules.MaxConsecutiveBlankLines != nil {
		blank := 0
		for i, ln := range fc.Metrics.LinesText {
			if strings.TrimSpace(ln) == "" {
				blank++
			} else {
				blank = 0
			}
			if blank > *rules.MaxConsecutiveBlankLines {
				violations = append(violations, Violation{
					RuleID:  "max_consecutive_blank_lines",
					Message: "连续空行超出上限",
					Path:    fc.Path,
					Line:    i + 1,
					Column:  1,
					Snippet: snippetLine(ln),
					Actual:  blank,
					Limit:   *rules.MaxConsecutiveBlankLines,
					Scope:   "line",
				})
			}
		}
	}

	for _, pr := range rules.ForbiddenPatterns {
		if strings.TrimSpace(pr.Pattern) == "" {
			continue
		}
		rx, err := compileRule(pr)
		if err != nil {
			errs = append(errs, fmt.Errorf("forbidden_patterns 编译失败：%w", err))
			continue
		}
		idxs := rx.FindAllStringIndex(normText, -1)
		for _, idx := range idxs {
			pos := textutil.LineAndColumnByOffset(fc.Metrics.LinesText, lineOffsets, idx[0])
			violations = append(violations, Violation{
				RuleID:  "forbidden_pattern",
				Message: "命中禁止模式",
				Path:    fc.Path,
				Line:    pos.Line,
				Column:  pos.Column,
				Snippet: textutil.SnippetByRune(normText, idx[0], contextChars),
				Actual:  normText[idx[0]:idx[1]],
				Limit:   pr.Pattern,
				Scope:   "line",
			})
		}
	}

	for _, pr := range rules.RequiredPatterns {
		if strings.TrimSpace(pr.Pattern) == "" {
			continue
		}
		rx, err := compileRule(pr)
		if err != nil {
			errs = append(errs, fmt.Errorf("required_patterns 编译失败：%w", err))
			continue
		}
		if !rx.MatchString(normText) {
			violations = append(violations, Violation{
				RuleID:  "required_pattern",
				Message: "缺少必需模式",
				Path:    fc.Path,
				Line:    0,
				Column:  0,
				Snippet: "",
				Actual:  "not_found",
				Limit:   pr.Pattern,
				Scope:   "file",
			})
		}
	}

	return violations, errs
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
