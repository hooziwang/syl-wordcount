package scan

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

var defaultIgnoreDirs = map[string]struct{}{
	".git":         {},
	".svn":         {},
	"node_modules": {},
	"vendor":       {},
	"dist":         {},
	"build":        {},
}

type Options struct {
	Paths          []string
	CWD            string
	FollowSymlinks bool
	IgnorePatterns []string
}

type GitIgnoreMatcher struct {
	Base     string
	Patterns []string
}

type ScanResult struct {
	Files  []string
	Errors []ScanError
}

type ScanError struct {
	Code   string
	Path   string
	Detail string
}

func Collect(opts Options) ScanResult {
	m := make(map[string]struct{})
	var errs []ScanError
	matchers := loadGitIgnoreMatchers(opts)

	for _, in := range opts.Paths {
		abs, err := filepath.Abs(in)
		if err != nil {
			errs = append(errs, ScanError{Code: "input_abs_failed", Path: in, Detail: err.Error()})
			continue
		}
		info, err := os.Lstat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				errs = append(errs, ScanError{Code: "input_path_not_found", Path: abs, Detail: "路径不存在"})
				continue
			}
			errs = append(errs, ScanError{Code: "input_stat_failed", Path: abs, Detail: err.Error()})
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 && !opts.FollowSymlinks {
			errs = append(errs, ScanError{Code: "symlink_skipped", Path: abs, Detail: "默认不跟随软链接"})
			continue
		}
		if info.IsDir() {
			walkDir(abs, opts, matchers, m, &errs)
			continue
		}
		if isIgnored(abs, false, opts, matchers) {
			continue
		}
		m[abs] = struct{}{}
	}

	files := make([]string, 0, len(m))
	for p := range m {
		files = append(files, p)
	}
	sort.Strings(files)
	return ScanResult{Files: files, Errors: errs}
}

func walkDir(root string, opts Options, matchers []GitIgnoreMatcher, out map[string]struct{}, errs *[]ScanError) {
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			*errs = append(*errs, ScanError{Code: "walk_error", Path: path, Detail: err.Error()})
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if _, ok := defaultIgnoreDirs[name]; ok {
				return fs.SkipDir
			}
			if isIgnored(path, true, opts, matchers) {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 && !opts.FollowSymlinks {
			return nil
		}
		if isIgnored(path, false, opts, matchers) {
			return nil
		}
		abs, aerr := filepath.Abs(path)
		if aerr != nil {
			*errs = append(*errs, ScanError{Code: "input_abs_failed", Path: path, Detail: aerr.Error()})
			return nil
		}
		out[abs] = struct{}{}
		return nil
	})
}

func loadGitIgnoreMatchers(opts Options) []GitIgnoreMatcher {
	uniq := map[string]struct{}{}
	var bases []string
	addBase := func(b string) {
		if b == "" {
			return
		}
		if _, ok := uniq[b]; ok {
			return
		}
		uniq[b] = struct{}{}
		bases = append(bases, b)
	}
	addBase(opts.CWD)
	for _, p := range opts.Paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil {
			continue
		}
		if info.IsDir() {
			addBase(abs)
		} else {
			addBase(filepath.Dir(abs))
		}
	}
	var ms []GitIgnoreMatcher
	for _, base := range bases {
		gip := filepath.Join(base, ".gitignore")
		b, err := os.ReadFile(gip)
		if err != nil {
			continue
		}
		patterns := make([]string, 0)
		for _, raw := range strings.Split(string(b), "\n") {
			p := strings.TrimSpace(raw)
			if p == "" || strings.HasPrefix(p, "#") || strings.HasPrefix(p, "!") {
				continue
			}
			p = strings.TrimPrefix(filepath.ToSlash(p), "/")
			if strings.HasSuffix(p, "/") {
				p = p + "**"
			}
			patterns = append(patterns, p)
			if !strings.Contains(p, "/") {
				patterns = append(patterns, "**/"+p)
			}
		}
		ms = append(ms, GitIgnoreMatcher{Base: base, Patterns: patterns})
	}
	return ms
}

func isIgnored(absPath string, isDir bool, opts Options, matchers []GitIgnoreMatcher) bool {
	for _, p := range opts.IgnorePatterns {
		ok, err := doublestar.Match(p, absPath)
		if err == nil && ok {
			return true
		}
		if opts.CWD != "" {
			rel, rerr := filepath.Rel(opts.CWD, absPath)
			if rerr == nil {
				ok, err := doublestar.Match(p, filepath.ToSlash(rel))
				if err == nil && ok {
					return true
				}
			}
		}
	}
	for _, m := range matchers {
		rel, err := filepath.Rel(m.Base, absPath)
		if err != nil {
			continue
		}
		if strings.HasPrefix(rel, "..") {
			continue
		}
		rel = filepath.ToSlash(rel)
		for _, p := range m.Patterns {
			ok, err := doublestar.Match(p, rel)
			if err == nil && ok {
				return true
			}
			if isDir {
				ok, err = doublestar.Match(p, rel+"/")
				if err == nil && ok {
					return true
				}
			}
		}
	}
	return false
}

func ValidateFormat(v string) error {
	if v != "ndjson" && v != "json" {
		return fmt.Errorf("不支持的输出格式：%s（仅支持 ndjson/json）", v)
	}
	return nil
}
