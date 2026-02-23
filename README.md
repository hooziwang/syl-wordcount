# syl-wordcount

面向 AI 流水线的文本统计与规则校验 CLI。

特点：默认 `NDJSON`、字段稳定、信息完整，适合直接喂给 LLM/Agent/CI。

## 安装

### macOS (Homebrew)

安装（首次/已 tap 过都可用）：

```bash
brew update && brew install hooziwang/tap/syl-wordcount
```

升级：

```bash
brew update && brew upgrade hooziwang/tap/syl-wordcount
```

如果提示 `No available formula`（本地 tap 索引过期）：

```bash
brew untap hooziwang/tap && brew install hooziwang/tap/syl-wordcount
```

### Windows (Scoop)

安装：

```powershell
scoop update; scoop bucket add hooziwang https://github.com/hooziwang/scoop-bucket.git; scoop install syl-wordcount
```

升级：

```powershell
scoop update; scoop update syl-wordcount
```

如果提示找不到应用（bucket 索引过期）：

```powershell
scoop bucket rm hooziwang; scoop bucket add hooziwang https://github.com/hooziwang/scoop-bucket.git; scoop update; scoop install syl-wordcount
```

### 本地源码构建

```bash
make
```

`make` 默认流程：`fmt -> test -> install`，会安装到 `GOBIN` 或 `GOPATH/bin`。

## 核心命令

### 1) 默认统计模式

```bash
syl-wordcount <path1> <path2> ...
```

### 2) 规则校验模式（check）

```bash
syl-wordcount check <path1> <path2> ... --config ./rules.yaml
```

注意：

- `syl-wordcount stats ...` 已移除，不再支持。
- 统计请直接用裸命令：`syl-wordcount <paths...>`。
- `check` 可两种方式提供规则：`--config` 文件，或 `SYL_WC_*` 环境变量。
- `check` 默认只输出 `violation/error`（加 `--all` 才输出 `pass`）。

## 快速上手示例

### 示例 1：统计一个文件

```bash
syl-wordcount ./docs/readme.md
```

### 示例 2：统计一个目录（默认递归）

```bash
syl-wordcount ./docs
```

### 示例 3：混合输入（目录 + 文件）

```bash
syl-wordcount ./docs ./README.md ./notes/todo.txt
```

### 示例 4：输出单个 JSON（不是 NDJSON）

```bash
syl-wordcount ./docs --format json
```

### 示例 5：统计模式默认包含 SHA256

```bash
syl-wordcount ./docs
```

### 示例 6：设置并发

```bash
syl-wordcount ./docs --jobs 8
```

### 示例 7：限制单文件最大处理大小

```bash
syl-wordcount ./docs --max-file-size 5MB
```

### 示例 8：执行规则校验

```bash
syl-wordcount check ./docs --config ./examples/config.example.yaml
```

### 示例 9：多路径校验

```bash
syl-wordcount check ./docs ./notes ./README.md --config ./rules.yaml
```

### 示例 9.1：check 全量输出（包含 pass）

```bash
syl-wordcount check ./docs --config ./rules.yaml --all
```

### 示例 10：输入路径不存在（看 error 事件 + 退出码）

```bash
syl-wordcount /no/such/path > /tmp/swc-missing.ndjson; echo $?
rg '"type":"error"' /tmp/swc-missing.ndjson
tail -n 1 /tmp/swc-missing.ndjson
```

### 示例 11：二进制文件会被跳过

```bash
syl-wordcount ./some-image.png > /tmp/swc-binary.ndjson; echo $?
rg 'skipped_binary_file' /tmp/swc-binary.ndjson
```

### 示例 12：超大文件会被跳过

```bash
syl-wordcount ./docs --max-file-size 1KB > /tmp/swc-large.ndjson; echo $?
rg 'skipped_large_file' /tmp/swc-large.ndjson
```

## 参数说明（常用）

- `--format ndjson|json`：输出格式，默认 `ndjson`
- `--jobs N`：并发任务数，默认 `min(8, CPU核数)`
- `--max-file-size 10MB`：单文件处理上限（超限会跳过并输出 error 事件）
- 软链接处理：内部固定为不跟随（无 `--follow-symlinks` 开关）
- 统计模式默认附带 `hash`（sha256），无需额外参数
- `--config /path/rules.yaml`：规则配置文件（`check` 可选；不传时尝试读取 `SYL_WC_*`）
- `--all`：仅 `check` 模式有效，输出全量事件（包含 `pass`）
- `-v, --version`：输出版本

## 输出格式与事件模型

固定 `type` 枚举：

- `meta`
- `file_stats`
- `pass`
- `violation`
- `error`
- `summary`

### NDJSON 输出示例（统计模式）

```json
{"type":"meta","tool":"syl-wordcount","mode":"stats","output_format":"ndjson"}
{"type":"file_stats","path":"/abs/path/a.txt","chars":120,"lines":8,"max_line_width":42,"encoding":"utf-8","line_ending":"lf","language_guess":"en","file_size":512,"hash":"<sha256>"}
{"type":"summary","total_files":1,"processed_files":1,"skipped_files":0,"violation_count":0,"error_count":0,"exit_code":0}
```

### NDJSON 输出示例（check 模式，含违规）

```json
{"type":"meta","tool":"syl-wordcount","mode":"check","config_path":"/abs/rules.yaml"}
{"type":"violation","rule_id":"max_line_width","path":"/abs/path/a.md","line":12,"column":81,"overflow_start_column":81,"line_end_column":103,"message":"行宽超出上限","snippet":"..."}
{"type":"violation","rule_id":"forbidden_pattern","path":"/abs/path/a.md","line":20,"column":5,"message":"命中禁止模式","actual":"TODO","limit":"TODO"}
{"type":"summary","total_files":3,"processed_files":3,"pass_count":2,"violation_count":2,"error_count":0,"exit_code":1,"rule_stats":{"max_line_width":{"violations":1,"files":1},"forbidden_pattern":{"violations":1,"files":1}}}
```

### 错误事件示例

```json
{"type":"error","code":"input_path_not_found","category":"input","path":"/abs/missing","detail":"路径不存在","next_action":"确认路径存在且拼写正确，再重试","fix_example":"syl-wordcount /path/to/input_dir","doc_key":"input.path_not_found","recoverable":true}
{"type":"error","code":"skipped_binary_file","category":"input","path":"/abs/a.png","detail":"识别为二进制文件，已跳过","next_action":"这是二进制文件，建议用规则只保留文本扩展名","fix_example":"SYL_WC_ALLOWED_EXTENSIONS=.md,.txt syl-wordcount check /path/to/input_dir","doc_key":"input.binary_skipped","recoverable":true}
{"type":"error","code":"decode_failed","category":"input","path":"/abs/a.txt","detail":"无法识别文本编码（支持 utf-8/gbk/gb18030）","next_action":"先把文件转成 utf-8/gbk/gb18030 之一，再执行","fix_example":"iconv -f gbk -t utf-8 input.txt -o output.txt && syl-wordcount output.txt","doc_key":"input.decode_failed","recoverable":true}
```

错误事件字段约定：

- `next_action`：下一步建议动作（给 AI 直接执行/生成修复命令）。
- `fix_example`：可参考的一行命令。
- `doc_key`：稳定键名，便于规则引擎做映射。
- `recoverable`：是否可通过修正输入/配置后重试。

## 规则配置详解

参考模板：`examples/config.example.yaml`

完整示例（可直接复制）：

```yaml
rules:
  min_chars: 10
  max_chars: 5000
  min_lines: 1
  max_lines: 200
  max_line_width: 100
  avg_line_width: 80
  max_file_size: "2MB"

  no_trailing_spaces: true
  no_tabs: true
  no_fullwidth_space: true
  max_consecutive_blank_lines: 2

  allowed_extensions:
    - ".md"
    - ".txt"

  ignore_patterns:
    - "**/.cache/**"
    - "**/*.log"

  forbidden_patterns:
    - pattern: "TODO"
      case_sensitive: true
    - pattern: "password\\s*=\\s*.+"
      case_sensitive: false

  required_patterns:
    - pattern: "版权"
      case_sensitive: true

  section_rules:
    - heading_contains: "xxx"
      rules:
        max_chars: 200
    - heading_contains: "yyy"
      rules:
        max_chars: 500
        max_lines: 20
```

### 规则说明（逐项）

| 规则键 | 含义 | 典型用途 | 对应环境变量 |
|---|---|---|---|
| `min_chars` | 文件最少字符数（rune） | 防止内容过短 | `SYL_WC_MIN_CHARS` |
| `max_chars` | 文件最多字符数（rune） | 控制文档篇幅 | `SYL_WC_MAX_CHARS` |
| `min_lines` | 文件最少行数（包含空行） | 防止空内容/过少内容 | `SYL_WC_MIN_LINES` |
| `max_lines` | 文件最多行数（包含空行） | 限制过长文档 | `SYL_WC_MAX_LINES` |
| `max_line_width` | 单行显示宽度上限 | 控制可读性、避免超宽行 | `SYL_WC_MAX_LINE_WIDTH` |
| `avg_line_width` | 平均行宽上限 | 控制整体排版密度 | `SYL_WC_AVG_LINE_WIDTH` |
| `max_file_size` | 文件体积上限（`KB/MB/GB`） | 限制超大文件 | `SYL_WC_MAX_FILE_SIZE` |
| `no_trailing_spaces` | 禁止行尾空白 | 保持文本整洁，减少 diff 噪音 | `SYL_WC_NO_TRAILING_SPACES` |
| `no_tabs` | 禁止制表符 `\\t` | 统一缩进策略 | `SYL_WC_NO_TABS` |
| `no_fullwidth_space` | 禁止全角空格 `U+3000` | 避免隐蔽排版问题 | `SYL_WC_NO_FULLWIDTH_SPACE` |
| `max_consecutive_blank_lines` | 连续空行上限 | 防止文档稀疏、断裂 | `SYL_WC_MAX_CONSECUTIVE_BLANK_LINES` |
| `allowed_extensions` | 允许检查的扩展名白名单 | 只检查目标文件类型 | `SYL_WC_ALLOWED_EXTENSIONS`（逗号分隔） |
| `ignore_patterns` | 额外忽略路径模式（glob） | 排除缓存/产物目录 | `SYL_WC_IGNORE_PATTERNS`（逗号分隔） |
| `forbidden_patterns` | 禁止出现的正则模式列表 | 拦截敏感词/占位词 | `SYL_WC_FORBIDDEN_PATTERNS`（大小写敏感）/`SYL_WC_FORBIDDEN_PATTERNS_I`（不敏感） |
| `required_patterns` | 必须出现的正则模式列表 | 强制必须声明/关键字段 | `SYL_WC_REQUIRED_PATTERNS`（大小写敏感）/`SYL_WC_REQUIRED_PATTERNS_I`（不敏感） |
| `section_rules` | 章节级规则列表（每条可独立规则） | 不同章节使用不同阈值 | `SYL_WC_SECTION_RULES`（JSON 数组） |

补充说明：

- `forbidden_patterns` 命中一次就记一次违规（不会只报第一条）。
- `required_patterns` 是“全部必须命中”（AND 关系），缺一条就报一条。
- 正则引擎是 Go 原生 `regexp`（RE2 语义）。
- `ignore_patterns` 使用 glob 语法（例如 `**/*.log`、`**/dist/**`）。
- `section_rules` 当前字段：`heading_contains`（必填）+ `rules`（章节规则子块，至少一条规则）。
- 运行模式是“全局规则 + 章节规则”并行：全局规则仍按整文件检查，章节规则只在命中章节内检查。
- `section_rules` 章节边界基于 Markdown 标题（`#` ~ `######`）：从命中的标题行开始，到下一个“同级或更高层级”标题之前；统计时不包含标题行本身。
- 环境变量模式下，`section_rules` 使用 `SYL_WC_SECTION_RULES` 传 JSON 数组。

章节规则示例（不同章节使用不同规则）：

```yaml
rules:
  # 全局规则（整文件）
  no_trailing_spaces: true

  # 章节规则（仅命中章节）
  section_rules:
    - heading_contains: "xxx"
      rules:
        max_chars: 200
    - heading_contains: "yyy"
      rules:
        max_chars: 500
        max_lines: 20
```

### 章节规则详细范例

示例 Markdown（`/path/to/doc.md`）：

```md
# 总览
这里是总览正文（全局规则会检查这里）。

## xxx-产品说明
这一段可能比较长，专门给 section_rules 做更严格限制。

## yyy-FAQ
这是另一个章节，通常允许更宽松的字数和行数。
```

范例 1：只约束 `xxx` 章节，最多 200 字

```yaml
rules:
  section_rules:
    - heading_contains: "xxx"
      rules:
        max_chars: 200
```

范例 2：`xxx` 和 `yyy` 使用不同阈值

```yaml
rules:
  section_rules:
    - heading_contains: "xxx"
      rules:
        max_chars: 200
        max_lines: 12
    - heading_contains: "yyy"
      rules:
        max_chars: 600
        max_lines: 40
```

范例 3：全局规则 + 章节规则同时生效

```yaml
rules:
  # 全局：全文件都检查
  no_tabs: true
  no_trailing_spaces: true

  # 章节：仅命中标题的章节检查
  section_rules:
    - heading_contains: "xxx"
      rules:
        max_chars: 200
```

范例 4：章节内做正则规则

```yaml
rules:
  section_rules:
    - heading_contains: "xxx"
      rules:
        forbidden_patterns:
          - pattern: "TODO"
            case_sensitive: true
        required_patterns:
          - pattern: "注意事项"
            case_sensitive: true
```

执行命令：

```bash
syl-wordcount check /path/to/doc.md --config /path/to/rules.yaml
```

行为说明：

- 全局规则和章节规则会并行执行，结果会同时出现在输出中。
- 同一章节如果命中多条 `section_rules`，这些规则会叠加生效。
- `section_rules` 只匹配标题文本，不匹配正文。
- 章节范围不包含标题行本身，只统计标题下面的正文内容。
- 如果 `section_rules` 某项缺少 `heading_contains` 或缺少 `rules`，会报配置错误。
- `section_rules[].rules` 可使用与全局规则相同的规则键（如 `max_chars`、`max_lines`、`forbidden_patterns` 等）。

### 配置中的环境变量

支持两种：

- `${VAR}`
- `${VAR:-default}`

说明：

- YAML 负责定义规则结构；环境变量负责在运行时注入阈值，这样不用改文件就能切换标准。
- 也可以完全不传 `--config`，直接使用 `SYL_WC_*` 作为规则来源。

示例：

```yaml
rules:
  max_lines: ${MAX_LINES:-200}
  max_line_width: ${MAX_WIDTH:-100}
  forbidden_patterns:
    - pattern: "${BAN_WORD}"
      case_sensitive: false
```

运行前设置变量：

```bash
export MAX_LINES=120 MAX_WIDTH=88 BAN_WORD=TODO
syl-wordcount check ./docs --config ./rules.yaml
```

纯环境变量规则（不传 `--config`）示例：

```bash
SYL_WC_MAX_LINE_WIDTH=110 \
SYL_WC_MAX_CHARS=8000 \
SYL_WC_NO_TABS=true \
SYL_WC_FORBIDDEN_PATTERNS=TODO,password \
syl-wordcount check /path/to/input_dir
```

更多纯环境变量示例：

```bash
# 1) 仅限制字符上限
SYL_WC_MAX_CHARS=2000 syl-wordcount check /path/to/input_dir
```

```bash
# 2) 排版规则组合
SYL_WC_MAX_LINE_WIDTH=100 \
SYL_WC_NO_TABS=true \
SYL_WC_NO_TRAILING_SPACES=true \
SYL_WC_MAX_CONSECUTIVE_BLANK_LINES=2 \
syl-wordcount check /path/to/input_dir
```

```bash
# 3) 大小写敏感/不敏感的正则
SYL_WC_FORBIDDEN_PATTERNS=TODO,password \
SYL_WC_REQUIRED_PATTERNS=版权,免责声明 \
syl-wordcount check /path/to/input_dir

SYL_WC_FORBIDDEN_PATTERNS_I=todo,password \
SYL_WC_REQUIRED_PATTERNS_I=copyright \
syl-wordcount check /path/to/input_dir
```

```bash
# 4) 扩展名白名单 + 忽略路径
SYL_WC_ALLOWED_EXTENSIONS=.md,.txt \
SYL_WC_IGNORE_PATTERNS='**/.git/**,**/node_modules/**,**/*.png' \
syl-wordcount check /path/to/input_dir
```

```bash
# 5) check 全量输出（包含 pass）
SYL_WC_MAX_CHARS=2000 syl-wordcount check /path/to/input_dir --all
```

```bash
# 6) 章节规则（JSON 数组）
SYL_WC_SECTION_RULES='[
  {"heading_contains":"xxx","rules":{"max_chars":200}},
  {"heading_contains":"yyy","rules":{"max_chars":500,"max_lines":20}}
]' \
syl-wordcount check /path/to/input_dir
```

```bash
# 7) 全局规则 + 章节规则（都用环境变量）
SYL_WC_MAX_CHARS=4000 \
SYL_WC_SECTION_RULES='[
  {"heading_contains":"xxx","rules":{"max_chars":200}},
  {"heading_contains":"yyy","rules":{"max_chars":500,"max_lines":20}}
]' \
syl-wordcount check /path/to/input_dir
```

`SYL_WC_SECTION_RULES` 格式说明：

- 必须是 JSON 数组。
- 每项至少包含：`heading_contains`（字符串）和 `rules`（对象）。
- `rules` 对象里的键与 YAML 规则键一致。
- 如果 JSON 非法、字段缺失，`check` 会返回配置错误（退出码 `4`）。

可用的环境变量前缀：`SYL_WC_*`。常用键：

- `SYL_WC_MIN_CHARS`, `SYL_WC_MAX_CHARS`
- `SYL_WC_MIN_LINES`, `SYL_WC_MAX_LINES`
- `SYL_WC_MAX_LINE_WIDTH`, `SYL_WC_AVG_LINE_WIDTH`
- `SYL_WC_MAX_FILE_SIZE`
- `SYL_WC_NO_TRAILING_SPACES`, `SYL_WC_NO_TABS`, `SYL_WC_NO_FULLWIDTH_SPACE`
- `SYL_WC_MAX_CONSECUTIVE_BLANK_LINES`
- `SYL_WC_ALLOWED_EXTENSIONS`（逗号分隔）
- `SYL_WC_IGNORE_PATTERNS`（逗号分隔）
- `SYL_WC_FORBIDDEN_PATTERNS`, `SYL_WC_FORBIDDEN_PATTERNS_I`（逗号分隔）
- `SYL_WC_REQUIRED_PATTERNS`, `SYL_WC_REQUIRED_PATTERNS_I`（逗号分隔）
- `SYL_WC_SECTION_RULES`（JSON 数组，章节规则）

## 退出码

- `0`：全部合格
- `1`：存在规则不合格
- `2`：参数错误
- `3`：输入错误（路径/读取/解码/跳过等）
- `4`：配置错误
- `5`：内部错误

建议：

- 在自动化里优先使用进程退出码（`$?` / `%ERRORLEVEL%`）作为最终判定。
- `summary.exit_code` 会与该进程退出码保持一致，便于纯 JSON 流水线读取。

## Shell/CI 典型用法

### 1) 只关心是否通过

```bash
syl-wordcount check ./docs --config ./rules.yaml > result.ndjson; echo $?
```

### 2) 校验失败时中断流水线

```bash
syl-wordcount check ./docs --config ./rules.yaml > result.ndjson || exit $?
```

### 3) 统计结果转成 JSON 文件

```bash
syl-wordcount ./docs --format json > stats.json
```

### 4) 统计并只提取 summary 行（NDJSON）

```bash
syl-wordcount ./docs | rg '"type":"summary"'
```

### 5) 只提取违规明细

```bash
syl-wordcount check ./docs --config ./rules.yaml > result.ndjson || true
rg '"type":"violation"' result.ndjson
```

### 6) 按 rule_id 聚合（快速看最常见问题）

```bash
syl-wordcount check ./docs --config ./rules.yaml > result.ndjson || true
rg '"type":"summary"' result.ndjson
```

`summary.rule_stats` 会给出每条规则的 `violations/files` 统计。

### 7) GitHub Actions 例子

```yaml
name: text-quality
on: [push, pull_request]
jobs:
  wc:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: brew tap hooziwang/tap && brew install syl-wordcount
      - run: syl-wordcount check . --config ./rules.yaml > syl-wordcount.ndjson
```

## 内部固定逻辑

- 默认递归扫描目录
- 路径统一输出绝对路径
- 自动识别文本/二进制
- 编码：UTF-8 优先，失败尝试 GBK/GB18030
- 字符数按 `rune`
- 行数 `lines` 包含空行；空文件为 `0` 行；末尾换行不会额外多算一行
- 最大行宽按显示宽度（CJK 宽字符）
- 列号 `column` 为 `rune` 列号（从 1 开始）
- tab 宽度按 4，按 tab stop 计算
- 软链接固定不跟随（不可配置）
- 默认启用 `.gitignore`，并内置忽略目录：`.git`、`.svn`、`node_modules`、`vendor`、`dist`、`build`
- check 模式必须有规则来源：`--config` 或 `SYL_WC_*` 环境变量（两者都没有会直接报错）
