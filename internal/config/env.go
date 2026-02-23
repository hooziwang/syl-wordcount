package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

const EnvPrefix = "SYL_WC_"

// LoadRulesForCheck 在 check 模式下加载规则：优先 --config，其次 SYL_WC_* 环境变量。
func LoadRulesForCheck(configPath string) (Rules, string, error) {
	if strings.TrimSpace(configPath) != "" {
		cfg, err := Load(configPath)
		if err != nil {
			return Rules{}, "", err
		}
		return cfg.Rules, configPath, nil
	}
	rules, ok, err := LoadRulesFromEnv(EnvPrefix)
	if err != nil {
		return Rules{}, "", err
	}
	if !ok {
		return Rules{}, "", fmt.Errorf("check 模式需要规则：请传 --config，或设置 %s* 环境变量", EnvPrefix)
	}
	return rules, "env://" + EnvPrefix + "*", nil
}

// LoadRulesFromEnv 从环境变量加载规则。
// 例如：SYL_WC_MAX_LINE_WIDTH=100, SYL_WC_MAX_CHARS=5000
// 章节规则可通过 SYL_WC_SECTION_RULES 传 JSON 数组。
func LoadRulesFromEnv(prefix string) (Rules, bool, error) {
	r := Rules{}
	has := false

	setIntPtr := func(key string, dst **int) error {
		v, ok := os.LookupEnv(prefix + key)
		if !ok {
			return nil
		}
		has = true
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fmt.Errorf("环境变量 %s%s 不是有效整数", prefix, key)
		}
		*dst = &n
		return nil
	}
	setBool := func(key string, dst *bool) error {
		v, ok := os.LookupEnv(prefix + key)
		if !ok {
			return nil
		}
		has = true
		b, err := parseBool(v)
		if err != nil {
			return fmt.Errorf("环境变量 %s%s 不是有效布尔值", prefix, key)
		}
		*dst = b
		return nil
	}
	setString := func(key string, dst *string) {
		v, ok := os.LookupEnv(prefix + key)
		if !ok {
			return
		}
		has = true
		*dst = strings.TrimSpace(v)
	}
	setList := func(key string, dst *[]string) {
		v, ok := os.LookupEnv(prefix + key)
		if !ok {
			return
		}
		has = true
		*dst = splitCSV(v)
	}
	setPatterns := func(key string, caseSensitive bool, dst *[]PatternRule) {
		v, ok := os.LookupEnv(prefix + key)
		if !ok {
			return
		}
		has = true
		for _, p := range splitCSV(v) {
			cs := caseSensitive
			*dst = append(*dst, PatternRule{Pattern: p, CaseSensitive: &cs})
		}
	}
	setSectionRules := func(key string, dst *[]SectionRule) error {
		v, ok := os.LookupEnv(prefix + key)
		if !ok {
			return nil
		}
		has = true
		raw := strings.TrimSpace(v)
		if raw == "" {
			*dst = nil
			return nil
		}
		var parsed []SectionRule
		if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
			return fmt.Errorf("环境变量 %s%s 不是有效 JSON：%w", prefix, key, err)
		}
		*dst = parsed
		return nil
	}

	if err := setIntPtr("MIN_CHARS", &r.MinChars); err != nil {
		return Rules{}, false, err
	}
	if err := setIntPtr("MAX_CHARS", &r.MaxChars); err != nil {
		return Rules{}, false, err
	}
	if err := setIntPtr("MIN_LINES", &r.MinLines); err != nil {
		return Rules{}, false, err
	}
	if err := setIntPtr("MAX_LINES", &r.MaxLines); err != nil {
		return Rules{}, false, err
	}
	if err := setIntPtr("MAX_LINE_WIDTH", &r.MaxLineWidth); err != nil {
		return Rules{}, false, err
	}
	if err := setIntPtr("AVG_LINE_WIDTH", &r.AvgLineWidth); err != nil {
		return Rules{}, false, err
	}
	if err := setIntPtr("MAX_CONSECUTIVE_BLANK_LINES", &r.MaxConsecutiveBlankLines); err != nil {
		return Rules{}, false, err
	}

	setString("MAX_FILE_SIZE", &r.MaxFileSize)
	setList("ALLOWED_EXTENSIONS", &r.AllowedExtensions)
	setList("IGNORE_PATTERNS", &r.IgnorePatterns)

	if err := setBool("NO_TRAILING_SPACES", &r.NoTrailingSpaces); err != nil {
		return Rules{}, false, err
	}
	if err := setBool("NO_TABS", &r.NoTabs); err != nil {
		return Rules{}, false, err
	}
	if err := setBool("NO_FULLWIDTH_SPACE", &r.NoFullwidthSpace); err != nil {
		return Rules{}, false, err
	}

	setPatterns("FORBIDDEN_PATTERNS", true, &r.ForbiddenPatterns)
	setPatterns("FORBIDDEN_PATTERNS_I", false, &r.ForbiddenPatterns)
	setPatterns("REQUIRED_PATTERNS", true, &r.RequiredPatterns)
	setPatterns("REQUIRED_PATTERNS_I", false, &r.RequiredPatterns)
	if err := setSectionRules("SECTION_RULES", &r.SectionRules); err != nil {
		return Rules{}, false, err
	}

	return r, has, nil
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func parseBool(v string) (bool, error) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "1", "true", "yes", "y", "on":
		return true, nil
	case "0", "false", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("invalid bool")
	}
}
