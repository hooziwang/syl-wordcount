package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type PatternRule struct {
	Pattern       string `yaml:"pattern" json:"pattern"`
	CaseSensitive *bool  `yaml:"case_sensitive" json:"case_sensitive"`
}

type SectionScopedRules struct {
	MinChars                 *int          `yaml:"min_chars" json:"min_chars"`
	MaxChars                 *int          `yaml:"max_chars" json:"max_chars"`
	MinLines                 *int          `yaml:"min_lines" json:"min_lines"`
	MaxLines                 *int          `yaml:"max_lines" json:"max_lines"`
	MaxLineWidth             *int          `yaml:"max_line_width" json:"max_line_width"`
	AvgLineWidth             *int          `yaml:"avg_line_width" json:"avg_line_width"`
	NoTrailingSpaces         bool          `yaml:"no_trailing_spaces" json:"no_trailing_spaces"`
	NoTabs                   bool          `yaml:"no_tabs" json:"no_tabs"`
	NoFullwidthSpace         bool          `yaml:"no_fullwidth_space" json:"no_fullwidth_space"`
	MaxConsecutiveBlankLines *int          `yaml:"max_consecutive_blank_lines" json:"max_consecutive_blank_lines"`
	ForbiddenPatterns        []PatternRule `yaml:"forbidden_patterns" json:"forbidden_patterns"`
	RequiredPatterns         []PatternRule `yaml:"required_patterns" json:"required_patterns"`
}

type SectionRule struct {
	HeadingContains string             `yaml:"heading_contains" json:"heading_contains"`
	Rules           SectionScopedRules `yaml:"rules" json:"rules"`
}

type Rules struct {
	MinChars                 *int          `yaml:"min_chars"`
	MaxChars                 *int          `yaml:"max_chars"`
	MinLines                 *int          `yaml:"min_lines"`
	MaxLines                 *int          `yaml:"max_lines"`
	MaxLineWidth             *int          `yaml:"max_line_width"`
	AvgLineWidth             *int          `yaml:"avg_line_width"`
	MaxFileSize              string        `yaml:"max_file_size"`
	NoTrailingSpaces         bool          `yaml:"no_trailing_spaces"`
	NoTabs                   bool          `yaml:"no_tabs"`
	NoFullwidthSpace         bool          `yaml:"no_fullwidth_space"`
	MaxConsecutiveBlankLines *int          `yaml:"max_consecutive_blank_lines"`
	ForbiddenPatterns        []PatternRule `yaml:"forbidden_patterns"`
	RequiredPatterns         []PatternRule `yaml:"required_patterns"`
	AllowedExtensions        []string      `yaml:"allowed_extensions"`
	IgnorePatterns           []string      `yaml:"ignore_patterns"`
	SectionRules             []SectionRule `yaml:"section_rules"`
}

type Config struct {
	Rules Rules `yaml:"rules"`
}

func Load(path string) (Config, error) {
	var cfg Config
	if strings.TrimSpace(path) == "" {
		return cfg, fmt.Errorf("配置文件路径为空")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("读取配置文件失败：%w", err)
	}
	expanded, err := expandEnv(string(b))
	if err != nil {
		return cfg, err
	}
	dec := yaml.NewDecoder(strings.NewReader(expanded))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("解析配置文件失败：%w", err)
	}
	return cfg, nil
}

var envExpr = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

func expandEnv(src string) (string, error) {
	var out strings.Builder
	last := 0
	for _, idx := range envExpr.FindAllStringSubmatchIndex(src, -1) {
		out.WriteString(src[last:idx[0]])
		name := src[idx[2]:idx[3]]
		hasDefault := idx[4] >= 0 && idx[5] >= 0
		defVal := ""
		if hasDefault && idx[6] >= 0 && idx[7] >= 0 {
			defVal = src[idx[6]:idx[7]]
		}
		if v, ok := os.LookupEnv(name); ok {
			out.WriteString(v)
		} else if hasDefault {
			out.WriteString(defVal)
		} else {
			return "", fmt.Errorf("配置中引用了未设置的环境变量：%s", name)
		}
		last = idx[1]
	}
	out.WriteString(src[last:])
	return out.String(), nil
}

func ParseSizeToBytes(s string) (int64, error) {
	v := strings.TrimSpace(strings.ToUpper(s))
	if v == "" {
		return 0, nil
	}
	units := []struct {
		U string
		M int64
	}{
		{"GB", 1024 * 1024 * 1024},
		{"MB", 1024 * 1024},
		{"KB", 1024},
		{"B", 1},
	}
	for _, unit := range units {
		if strings.HasSuffix(v, unit.U) {
			n := strings.TrimSpace(strings.TrimSuffix(v, unit.U))
			f, err := strconv.ParseFloat(n, 64)
			if err != nil {
				return 0, fmt.Errorf("无效大小值：%s", s)
			}
			return int64(f * float64(unit.M)), nil
		}
	}
	// 纯数字按字节
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("无效大小值：%s", s)
	}
	return n, nil
}
