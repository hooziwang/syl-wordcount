# syl-wordcount

面向 AI 流水线的文本统计与规则校验 CLI。

特点：默认 `NDJSON`、字段稳定、信息完整，适合直接喂给 LLM/Agent/CI。

## 安装

### macOS (Homebrew)

```bash
brew tap hooziwang/tap && brew install syl-wordcount
```

### Windows (Scoop)

```powershell
scoop bucket add hooziwang https://github.com/hooziwang/scoop-bucket.git; scoop install syl-wordcount
```

### 本地源码构建

```bash
make
```

`make` 默认流程：`fmt -> test -> install`，会安装到 `GOBIN` 或 `GOPATH/bin`。

## 核心命令

### 1) 默认统计模式（stats）

```bash
syl-wordcount <path1> <path2> ...
```

等价于：

```bash
syl-wordcount stats <path1> <path2> ...
```

### 2) 规则校验模式（check）

```bash
syl-wordcount check <path1> <path2> ... --config ./rules.yaml
```

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

### 示例 5：附加 SHA256（仅统计模式）

```bash
syl-wordcount ./docs --with-hash sha256
```

### 示例 6：设置并发

```bash
syl-wordcount ./docs --jobs 8
```

### 示例 7：限制单文件最大处理大小

```bash
syl-wordcount ./docs --max-file-size 5MB
```

### 示例 8：跟随软链接

```bash
syl-wordcount ./docs --follow-symlinks
```

### 示例 9：执行规则校验

```bash
syl-wordcount check ./docs --config ./examples/config.example.yaml
```

### 示例 10：多路径校验

```bash
syl-wordcount check ./docs ./notes ./README.md --config ./rules.yaml
```

## 参数说明（常用）

- `--format ndjson|json`：输出格式，默认 `ndjson`
- `--jobs N`：并发任务数，默认 `min(8, CPU核数)`
- `--follow-symlinks`：是否跟随软链接（默认不跟随）
- `--max-file-size 10MB`：单文件处理上限（超限会跳过并输出 error 事件）
- `--with-hash sha256`：统计模式附带文件哈希
- `--config /path/rules.yaml`：规则配置文件（`check` 必填）
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
{"type":"file_stats","path":"/abs/path/a.txt","chars":120,"lines":8,"max_line_width":42,"encoding":"utf-8","line_ending":"lf","language_guess":"en","file_size":512}
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
{"type":"error","code":"input_path_not_found","category":"input","path":"/abs/missing","detail":"路径不存在"}
{"type":"error","code":"skipped_binary_file","category":"input","path":"/abs/a.png","detail":"识别为二进制文件，已跳过"}
{"type":"error","code":"decode_failed","category":"input","path":"/abs/a.txt","detail":"无法识别文本编码（支持 utf-8/gbk/gb18030）"}
```

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
```

### 配置中的环境变量

支持两种：

- `${VAR}`
- `${VAR:-default}`

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

## 退出码

- `0`：全部合格
- `1`：存在规则不合格
- `2`：参数错误
- `3`：输入错误（路径/读取/解码/跳过等）
- `4`：配置错误
- `5`：内部错误

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

### 5) GitHub Actions 例子

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

## 设计口径（已固定）

- 默认递归扫描目录
- 路径统一输出绝对路径
- 自动识别文本/二进制
- 编码：UTF-8 优先，失败尝试 GBK/GB18030
- 字符数按 `rune`
- 最大行宽按显示宽度（CJK 宽字符）
- tab 宽度按 4，按 tab stop 计算
- 默认启用 `.gitignore`，并内置忽略目录：`.git`、`.svn`、`node_modules`、`vendor`、`dist`、`build`
- check 模式必须传 `--config`

## 发布

项目已包含：

- `.github/workflows/release.yml`
- `.goreleaser.yml`

打 tag 即发布：

```bash
git tag -a v0.1.0 -m "release v0.1.0" && git push origin v0.1.0
```

会自动发布到：

- GitHub Releases
- `hooziwang/homebrew-tap`
- `hooziwang/scoop-bucket`
