package app

type errorHint struct {
	NextAction  string
	FixExample  string
	DocKey      string
	Recoverable bool
}

func buildErrorEvent(category, code, path, detail string) map[string]any {
	h := hintByCode(code)
	return map[string]any{
		"type":        "error",
		"code":        code,
		"category":    category,
		"path":        path,
		"detail":      detail,
		"next_action": h.NextAction,
		"fix_example": h.FixExample,
		"doc_key":     h.DocKey,
		"recoverable": h.Recoverable,
	}
}

func hintByCode(code string) errorHint {
	switch code {
	case "input_path_not_found":
		return errorHint{
			NextAction:  "确认路径存在且拼写正确，再重试",
			FixExample:  "syl-wordcount /path/to/input_dir",
			DocKey:      "input.path_not_found",
			Recoverable: true,
		}
	case "input_abs_failed", "input_stat_failed", "walk_error", "file_stat_failed", "file_read_failed":
		return errorHint{
			NextAction:  "检查路径权限和可读性，必要时更换输入目录",
			FixExample:  "chmod -R +r /path/to/input_dir && syl-wordcount /path/to/input_dir",
			DocKey:      "input.path_access",
			Recoverable: true,
		}
	case "symlink_skipped":
		return errorHint{
			NextAction:  "该工具固定不跟随软链接，请改为传真实路径",
			FixExample:  "syl-wordcount /real/path/to/input_dir",
			DocKey:      "input.symlink_skipped",
			Recoverable: true,
		}
	case "skipped_large_file":
		return errorHint{
			NextAction:  "增大 --max-file-size，或排除该大文件",
			FixExample:  "syl-wordcount /path/to/input_dir --max-file-size 50MB",
			DocKey:      "input.max_file_size",
			Recoverable: true,
		}
	case "skipped_binary_file":
		return errorHint{
			NextAction:  "这是二进制文件，建议用规则只保留文本扩展名",
			FixExample:  "SYL_WC_ALLOWED_EXTENSIONS=.md,.txt syl-wordcount check /path/to/input_dir",
			DocKey:      "input.binary_skipped",
			Recoverable: true,
		}
	case "decode_failed":
		return errorHint{
			NextAction:  "先把文件转成 utf-8/gbk/gb18030 之一，再执行",
			FixExample:  "iconv -f gbk -t utf-8 input.txt -o output.txt && syl-wordcount output.txt",
			DocKey:      "input.decode_failed",
			Recoverable: true,
		}
	case "rule_eval_error":
		return errorHint{
			NextAction:  "检查规则配置是否正确（尤其正则表达式）",
			FixExample:  "syl-wordcount check /path/to/input_dir --config /path/to/rules.yaml",
			DocKey:      "check.rule_eval_error",
			Recoverable: true,
		}
	default:
		return errorHint{
			NextAction:  "根据 detail 修正输入或配置后重试",
			FixExample:  "syl-wordcount --help",
			DocKey:      "general.error",
			Recoverable: true,
		}
	}
}
