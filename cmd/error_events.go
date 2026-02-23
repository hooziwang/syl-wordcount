package cmd

import (
	"io"
	"strings"

	"syl-wordcount/internal/output"
)

type cliErrorHint struct {
	NextAction  string
	FixExample  string
	DocKey      string
	Recoverable bool
}

func writeCLIError(w io.Writer, format string, mode string, args []string, code, category, path, detail string, exitCode int) {
	h := cliHintByCode(code)
	events := []map[string]any{
		{
			"type":          "meta",
			"tool":          "syl-wordcount",
			"version":       Version,
			"mode":          mode,
			"args":          args,
			"output_format": format,
		},
		{
			"type":        "error",
			"code":        code,
			"category":    category,
			"path":        path,
			"detail":      detail,
			"next_action": h.NextAction,
			"fix_example": h.FixExample,
			"doc_key":     h.DocKey,
			"recoverable": h.Recoverable,
		},
		{
			"type":            "summary",
			"mode":            mode,
			"total_files":     0,
			"processed_files": 0,
			"skipped_files":   0,
			"pass_count":      0,
			"violation_count": 0,
			"error_count":     1,
			"exit_code":       exitCode,
		},
	}
	_ = output.Write(w, normalizeFormat(format), events)
}

func normalizeFormat(format string) string {
	if format == "json" {
		return "json"
	}
	return "ndjson"
}

func detectFormatFromArgs(args []string) string {
	format := "ndjson"
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		if a == "--format" {
			if i+1 < len(args) {
				return normalizeFormat(args[i+1])
			}
			continue
		}
		if strings.HasPrefix(a, "--format=") {
			return normalizeFormat(strings.TrimPrefix(a, "--format="))
		}
	}
	return format
}

func cliHintByCode(code string) cliErrorHint {
	switch code {
	case "arg_missing_paths":
		return cliErrorHint{
			NextAction:  "至少传一个文件或目录路径",
			FixExample:  "syl-wordcount /path/to/input_dir",
			DocKey:      "arg.missing_paths",
			Recoverable: true,
		}
	case "invalid_output_format":
		return cliErrorHint{
			NextAction:  "把 --format 改为 ndjson 或 json",
			FixExample:  "syl-wordcount /path/to/input_dir --format ndjson",
			DocKey:      "arg.invalid_output_format",
			Recoverable: true,
		}
	case "invalid_max_file_size":
		return cliErrorHint{
			NextAction:  "把 --max-file-size 改成合法大小（如 10MB）",
			FixExample:  "syl-wordcount /path/to/input_dir --max-file-size 20MB",
			DocKey:      "arg.invalid_max_file_size",
			Recoverable: true,
		}
	case "invalid_input_paths":
		return cliErrorHint{
			NextAction:  "检查输入路径是否为空、是否可解析为绝对路径",
			FixExample:  "syl-wordcount /path/to/input_dir /path/to/file.md",
			DocKey:      "arg.invalid_input_paths",
			Recoverable: true,
		}
	case "check_rules_missing":
		return cliErrorHint{
			NextAction:  "check 必须提供规则：传 --config 或设置 SYL_WC_* 环境变量",
			FixExample:  "SYL_WC_MAX_CHARS=2000 syl-wordcount check /path/to/input_dir",
			DocKey:      "check.rules_missing",
			Recoverable: true,
		}
	case "check_config_invalid":
		return cliErrorHint{
			NextAction:  "修正规则配置文件内容后重试",
			FixExample:  "syl-wordcount check /path/to/input_dir --config /path/to/rules.yaml",
			DocKey:      "check.config_invalid",
			Recoverable: true,
		}
	case "cwd_failed":
		return cliErrorHint{
			NextAction:  "确认当前工作目录可访问，或切换到可访问目录",
			FixExample:  "cd /path/to/workspace && syl-wordcount /path/to/input_dir",
			DocKey:      "runtime.cwd_failed",
			Recoverable: true,
		}
	case "output_write_failed":
		return cliErrorHint{
			NextAction:  "检查输出管道或重定向目标是否可写",
			FixExample:  "syl-wordcount /path/to/input_dir > result.ndjson",
			DocKey:      "runtime.output_write_failed",
			Recoverable: true,
		}
	case "unknown_command":
		return cliErrorHint{
			NextAction:  "确认命令拼写，或查看帮助",
			FixExample:  "syl-wordcount --help",
			DocKey:      "arg.unknown_command",
			Recoverable: true,
		}
	default:
		return cliErrorHint{
			NextAction:  "根据 detail 修正参数或配置后重试",
			FixExample:  "syl-wordcount --help",
			DocKey:      "general.error",
			Recoverable: true,
		}
	}
}
