package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"

	"syl-wordcount/internal/config"
	"syl-wordcount/internal/scan"
	"syl-wordcount/internal/textutil"
)

type fileResult struct {
	Path         string
	Events       []map[string]any
	HasViolation bool
	HasInputErr  bool
	Skipped      bool
	Processed    bool
	RuleHit      map[string]struct{}
}

func DefaultJobs() int {
	n := runtime.NumCPU()
	if n > 8 {
		return 8
	}
	if n < 1 {
		return 1
	}
	return n
}

func Run(opts Options) (Result, error) {
	res := Result{Events: make([]map[string]any, 0)}
	if opts.Jobs <= 0 {
		opts.Jobs = DefaultJobs()
	}
	if opts.MaxFileSizeBytes <= 0 {
		opts.MaxFileSizeBytes = 10 * 1024 * 1024
	}

	var cfg RuntimeConfig
	configPathForMeta := opts.ConfigPath
	if opts.Mode == ModeCheck {
		rules, source, err := config.LoadRulesForCheck(opts.ConfigPath)
		if err != nil {
			return res, &ConfigErr{Msg: err.Error()}
		}
		cfg = RuntimeConfig{Rules: rules}
		configPathForMeta = source
	} else if strings.TrimSpace(opts.ConfigPath) != "" {
		loaded, err := config.Load(opts.ConfigPath)
		if err != nil {
			return res, &ConfigErr{Msg: err.Error()}
		}
		cfg = RuntimeConfig{Rules: loaded.Rules}
	}

	meta := map[string]any{
		"type":             "meta",
		"tool":             "syl-wordcount",
		"version":          opts.Version,
		"mode":             string(opts.Mode),
		"cwd":              opts.CWD,
		"args":             opts.Args,
		"config_path":      configPathForMeta,
		"output_format":    opts.Format,
		"follow_symlinks":  false,
		"max_file_size":    opts.MaxFileSizeBytes,
		"exit_code_policy": map[string]int{"ok": 0, "violation": 1, "arg_error": 2, "input_error": 3, "config_error": 4, "internal_error": 5},
	}
	res.Events = append(res.Events, meta)

	scanRes := scan.Collect(scan.Options{
		Paths:          opts.Paths,
		CWD:            opts.CWD,
		FollowSymlinks: false,
		IgnorePatterns: cfg.Rules.IgnorePatterns,
	})
	for _, se := range scanRes.Errors {
		res.Events = append(res.Events, buildErrorEvent("input", se.Code, se.Path, se.Detail))
		res.HasInputErr = true
	}

	paths := scanRes.Files
	res.Summary.TotalFiles = len(paths)
	if len(paths) == 0 {
		for _, e := range res.Events {
			t, _ := e["type"].(string)
			if t == "error" {
				res.Summary.Errors++
			}
		}
		res.Events = append(res.Events, buildSummary(opts.Mode, res.Summary, decideExitCode(res)))
		return res, nil
	}

	jobs := opts.Jobs
	if jobs > len(paths) {
		jobs = len(paths)
	}
	in := make(chan string)
	out := make(chan fileResult, len(paths))
	wg := sync.WaitGroup{}

	for i := 0; i < jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range in {
				out <- processFile(p, opts, cfg)
			}
		}()
	}

	for _, p := range paths {
		in <- p
	}
	close(in)
	wg.Wait()
	close(out)

	byPath := map[string]fileResult{}
	for fr := range out {
		byPath[fr.Path] = fr
	}

	ruleFiles := map[string]map[string]struct{}{}
	sort.Strings(paths)
	for _, p := range paths {
		fr := byPath[p]
		res.Events = append(res.Events, fr.Events...)
		if fr.Processed {
			res.Summary.Processed++
		}
		if fr.Skipped {
			res.Summary.Skipped++
		}
		if fr.HasViolation {
			res.HasViolation = true
		}
		if fr.HasInputErr {
			res.HasInputErr = true
		}
		for rid := range fr.RuleHit {
			if _, ok := ruleFiles[rid]; !ok {
				ruleFiles[rid] = map[string]struct{}{}
			}
			ruleFiles[rid][p] = struct{}{}
		}
	}

	res.Summary.RuleStats = map[string]RuleStats{}
	for _, e := range res.Events {
		t, _ := e["type"].(string)
		switch t {
		case "violation":
			res.Summary.Violations++
			rid, _ := e["rule_id"].(string)
			rs := res.Summary.RuleStats[rid]
			rs.Violations++
			res.Summary.RuleStats[rid] = rs
		case "pass":
			res.Summary.PassCount++
		case "error":
			res.Summary.Errors++
		}
	}
	for rid, files := range ruleFiles {
		rs := res.Summary.RuleStats[rid]
		rs.Files = len(files)
		res.Summary.RuleStats[rid] = rs
	}

	res.Events = append(res.Events, buildSummary(opts.Mode, res.Summary, decideExitCode(res)))
	return res, nil
}

func processFile(path string, opts Options, cfg RuntimeConfig) fileResult {
	fr := fileResult{Path: path, Events: make([]map[string]any, 0), RuleHit: map[string]struct{}{}}
	info, err := os.Stat(path)
	if err != nil {
		fr.HasInputErr = true
		fr.Events = append(fr.Events, buildErrorEvent("input", "file_stat_failed", path, err.Error()))
		return fr
	}
	if info.IsDir() {
		return fr
	}

	if info.Size() > opts.MaxFileSizeBytes {
		fr.Skipped = true
		fr.Events = append(fr.Events, buildErrorEvent("input", "skipped_large_file", path, fmt.Sprintf("文件大小 %d 超过上限 %d", info.Size(), opts.MaxFileSizeBytes)))
		return fr
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fr.HasInputErr = true
		fr.Events = append(fr.Events, buildErrorEvent("input", "file_read_failed", path, err.Error()))
		return fr
	}

	sample := data
	if len(sample) > 8192 {
		sample = sample[:8192]
	}
	if textutil.DetectBinary(sample) {
		fr.Skipped = true
		fr.Events = append(fr.Events, buildErrorEvent("input", "skipped_binary_file", path, "识别为二进制文件，已跳过"))
		return fr
	}

	decoded, err := textutil.Decode(data)
	if err != nil {
		fr.Skipped = true
		fr.Events = append(fr.Events, buildErrorEvent("input", "decode_failed", path, err.Error()))
		return fr
	}
	metrics := textutil.ComputeMetrics(decoded.Text)

	if opts.Mode == ModeStats {
		ev := map[string]any{
			"type":           "file_stats",
			"path":           path,
			"status":         "ok",
			"encoding":       decoded.Encoding,
			"file_size":      len(data),
			"hash":           textutil.HashSHA256(data),
			"line_ending":    metrics.LineEnding,
			"language_guess": metrics.Language,
			"chars":          metrics.Chars,
			"lines":          metrics.Lines,
			"max_line_width": metrics.MaxLineWidth,
		}
		fr.Events = append(fr.Events, ev)
		fr.Processed = true
		return fr
	}

	fc := FileContent{Path: path, Data: data, Text: decoded.Text, Encoding: decoded.Encoding, Metrics: metrics}
	violations, verrs := EvaluateRules(fc, cfg.Rules)
	for _, e := range verrs {
		fr.HasInputErr = true
		fr.Events = append(fr.Events, buildErrorEvent("config", "rule_eval_error", path, e.Error()))
	}

	if len(violations) == 0 && len(verrs) == 0 {
		fr.Events = append(fr.Events, map[string]any{
			"type": "pass",
			"path": path,
		})
		fr.Processed = true
		return fr
	}
	if len(violations) > 0 {
		fr.HasViolation = true
		for _, v := range violations {
			fr.RuleHit[v.RuleID] = struct{}{}
			fr.Events = append(fr.Events, map[string]any{
				"type":                  "violation",
				"rule_id":               v.RuleID,
				"message":               v.Message,
				"path":                  v.Path,
				"line":                  v.Line,
				"column":                v.Column,
				"overflow_start_column": v.OverflowStartColumn,
				"line_end_column":       v.LineEndColumn,
				"snippet":               v.Snippet,
				"actual":                v.Actual,
				"limit":                 v.Limit,
				"scope":                 v.Scope,
			})
		}
	}
	fr.Processed = true
	return fr
}

func buildSummary(mode Mode, s Summary, exitCode int) map[string]any {
	m := map[string]any{
		"type":            "summary",
		"mode":            string(mode),
		"total_files":     s.TotalFiles,
		"processed_files": s.Processed,
		"skipped_files":   s.Skipped,
		"pass_count":      s.PassCount,
		"violation_count": s.Violations,
		"error_count":     s.Errors,
		"exit_code":       exitCode,
	}
	if len(s.RuleStats) > 0 {
		m["rule_stats"] = s.RuleStats
	}
	return m
}

type ConfigErr struct{ Msg string }

func (e *ConfigErr) Error() string { return e.Msg }

type ArgErr struct{ Msg string }

func (e *ArgErr) Error() string { return e.Msg }

func decideExitCode(res Result) int {
	if res.HasInternalErr {
		return 5
	}
	if res.HasConfigErr {
		return 4
	}
	if res.HasInputErr {
		return 3
	}
	if res.HasViolation {
		return 1
	}
	return 0
}

func NormalizePaths(paths []string, cwd string) []string {
	out := make([]string, 0, len(paths))
	seen := map[string]struct{}{}
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	sort.Strings(out)
	return out
}
