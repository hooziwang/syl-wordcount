package cmd

import "strings"

func rootLongHelp() string {
	return strings.TrimSpace(`
面向 AI 的文本统计与规则校验 CLI。

两种使用方式（AI 首选）：
1. 统计字数（默认模式）
   - 命令：syl-wordcount <path...>
   - 输出：file_stats 事件（chars / lines / max_line_width / hash）
2. 规则校验（check 模式）
   - 命令：syl-wordcount check <path...> --config rules.yaml
   - 或：仅用 SYL_WC_* 环境变量
   - 输出：violation/error 事件（默认隐藏 pass；可用 --all 输出 pass）

输入与扫描：
- 支持多个文件、多个目录、文件+目录混合
- 目录默认递归
- 固定不跟随软链接
- 默认忽略目录：.git/.svn/node_modules/vendor/dist/build
- 默认忽略文件：.DS_Store

输出模型（NDJSON 默认）：
- meta
- file_stats
- pass
- violation
- error
- summary

error 事件（给 AI 直接执行）：
- code/category/path/detail
- next_action（下一步）
- fix_example（示例命令）
- doc_key（稳定键）
- recoverable（是否可重试）

退出码：
- 0 通过
- 1 存在违规
- 2 参数错误
- 3 输入错误
- 4 配置错误
- 5 内部错误

完整规则说明：请看 ` + "`syl-wordcount check --help`" + `
`)
}

func rootExampleHelp() string {
	return strings.TrimSpace(`
  # 方式 1：统计字数（单文件）
  syl-wordcount /path/to/a.md

  # 方式 1：统计字数（目录）
  syl-wordcount /path/to/docs

  # 方式 1：统计字数（目录 + 文件）
  syl-wordcount /path/to/docs /path/to/README.md

  # 方式 1：改成 JSON 输出
  syl-wordcount /path/to/docs --format json

  # 方式 2：规则校验（配置文件）
  syl-wordcount check /path/to/docs --config /path/to/rules.yaml

  # 方式 2：规则校验（纯环境变量）
  SYL_WC_MAX_CHARS=2000 SYL_WC_NO_TABS=true syl-wordcount check /path/to/docs

  # 方式 2：校验时输出 pass + violation + error
  syl-wordcount check /path/to/docs --config /path/to/rules.yaml --all
`)
}

func checkLongHelp() string {
	return strings.TrimSpace(`
规则校验模式：按规则检查文本并输出 violation/error（可定位到行列与片段）。

规则来源（至少一种）：
1. --config /path/to/rules.yaml
2. SYL_WC_* 环境变量（不传 --config 也可以）

执行模型：
- 全局规则：作用于整文件
- 章节规则：section_rules，作用于命中标题的章节
- 全局规则 + section_rules 并行执行，结果会同时输出

section_rules 说明：
- 字段：heading_contains + rules
- 匹配方式：标题文本包含 heading_contains 即命中
- 章节边界：从命中标题起，到下一个同级或更高层级标题前
- 作用域标记：违规事件中 scope=section

规则说明（全部）：
1. min_chars
   - 含义：最少字符数（rune）
   - 环境变量：SYL_WC_MIN_CHARS
2. max_chars
   - 含义：最多字符数（rune）
   - 环境变量：SYL_WC_MAX_CHARS
3. min_lines
   - 含义：最少行数（包含空行）
   - 环境变量：SYL_WC_MIN_LINES
4. max_lines
   - 含义：最多行数（包含空行）
   - 环境变量：SYL_WC_MAX_LINES
5. max_line_width
   - 含义：单行显示宽度上限
   - 环境变量：SYL_WC_MAX_LINE_WIDTH
6. avg_line_width
   - 含义：平均行宽上限
   - 环境变量：SYL_WC_AVG_LINE_WIDTH
7. max_file_size
   - 含义：文件体积上限（如 10MB）
   - 环境变量：SYL_WC_MAX_FILE_SIZE
8. no_trailing_spaces
   - 含义：禁止行尾空白
   - 环境变量：SYL_WC_NO_TRAILING_SPACES
9. no_tabs
   - 含义：禁止制表符 \t
   - 环境变量：SYL_WC_NO_TABS
10. no_fullwidth_space
   - 含义：禁止全角空格 U+3000
   - 环境变量：SYL_WC_NO_FULLWIDTH_SPACE
11. max_consecutive_blank_lines
   - 含义：连续空行最大数量
   - 环境变量：SYL_WC_MAX_CONSECUTIVE_BLANK_LINES
12. forbidden_patterns
   - 含义：禁止出现的正则模式（命中即违规）
   - 环境变量：
     - SYL_WC_FORBIDDEN_PATTERNS（大小写敏感，逗号分隔）
     - SYL_WC_FORBIDDEN_PATTERNS_I（大小写不敏感，逗号分隔）
13. required_patterns
   - 含义：必须出现的正则模式（全部都要命中）
   - 环境变量：
     - SYL_WC_REQUIRED_PATTERNS（大小写敏感，逗号分隔）
     - SYL_WC_REQUIRED_PATTERNS_I（大小写不敏感，逗号分隔）
14. allowed_extensions
   - 含义：允许检查的扩展名白名单
   - 环境变量：SYL_WC_ALLOWED_EXTENSIONS（逗号分隔）
15. ignore_patterns
   - 含义：额外忽略路径模式（glob）
   - 环境变量：SYL_WC_IGNORE_PATTERNS（逗号分隔）
16. section_rules
   - 含义：章节级规则列表（每条可独立配置）
   - 环境变量：SYL_WC_SECTION_RULES（JSON 数组）

section_rules.rules 可用子规则：
- min_chars / max_chars
- min_lines / max_lines
- max_line_width / avg_line_width
- no_trailing_spaces / no_tabs / no_fullwidth_space
- max_consecutive_blank_lines
- forbidden_patterns / required_patterns

注意：
- check 如果没有任何规则来源，会返回配置错误（退出码 4）
- 正则引擎为 Go RE2 语义
- ignore_patterns 使用 glob 语法（如 **/*.log）
`)
}

func checkExampleHelp() string {
	return strings.TrimSpace(`
  # 1) 配置文件校验
  syl-wordcount check /path/to/docs --config /path/to/rules.yaml

  # 2) 纯环境变量校验
  SYL_WC_MAX_LINE_WIDTH=100 SYL_WC_NO_TABS=true syl-wordcount check /path/to/docs

  # 3) 全局 + 章节规则（环境变量）
  SYL_WC_MAX_CHARS=4000 \
  SYL_WC_SECTION_RULES='[
    {"heading_contains":"xxx","rules":{"max_chars":200}},
    {"heading_contains":"yyy","rules":{"max_chars":500,"max_lines":20}}
  ]' \
  syl-wordcount check /path/to/docs

  # 4) 全量输出（包含 pass）
  syl-wordcount check /path/to/docs --config /path/to/rules.yaml --all
`)
}
