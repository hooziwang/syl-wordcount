package app

import "syl-wordcount/internal/config"

type Mode string

const (
	ModeStats Mode = "stats"
	ModeCheck Mode = "check"
)

type Options struct {
	Mode             Mode
	Paths            []string
	CWD              string
	ConfigPath       string
	Format           string
	Jobs             int
	MaxFileSizeBytes int64
	Version          string
	Args             []string
}

type RuleStats struct {
	Violations int `json:"violations"`
	Files      int `json:"files"`
}

type Summary struct {
	TotalFiles int                  `json:"total_files"`
	Processed  int                  `json:"processed_files"`
	Skipped    int                  `json:"skipped_files"`
	PassCount  int                  `json:"pass_count"`
	Violations int                  `json:"violation_count"`
	Errors     int                  `json:"error_count"`
	RuleStats  map[string]RuleStats `json:"rule_stats,omitempty"`
}

type Result struct {
	Events         []map[string]any
	Summary        Summary
	HasViolation   bool
	HasInputErr    bool
	HasConfigErr   bool
	HasInternalErr bool
}

type RuntimeConfig struct {
	Rules config.Rules
}
