# syl-wordcount

面向 AI 流水线的文本统计与规则校验工具。

默认输出 `NDJSON`，事件完整、字段稳定，适合直接喂给 LLM/Agent/CI。

## 安装

### Homebrew（macOS）

```bash
brew tap hooziwang/tap && brew install syl-wordcount
```

### Scoop（Windows）

```powershell
scoop bucket add hooziwang https://github.com/hooziwang/scoop-bucket.git; scoop install syl-wordcount
```

### 本地编译

```bash
make
```

`make` 默认执行：`fmt -> test -> install`，并安装到 Go bin 目录。

## 命令

### 统计模式（默认）

```bash
syl-wordcount <path1> <path2> ...
```

### 规则校验模式

```bash
syl-wordcount check <path1> <path2> ... --config ./rules.yaml
```

## 常用参数

- `--format ndjson|json`：输出格式，默认 `ndjson`
- `--jobs N`：并发数，默认 `min(8, CPU核数)`
- `--follow-symlinks`：是否跟随软链接（默认不跟随）
- `--max-file-size 10MB`：单文件处理上限，超出跳过并输出错误事件
- `--with-hash sha256`：统计模式附带文件哈希
- `--config /path/rules.yaml`：校验配置文件（`check` 必填）
- `-v, --version`：版本信息

## 输出事件类型

固定 `type` 枚举：

- `meta`
- `file_stats`
- `pass`
- `violation`
- `error`
- `summary`

## 退出码

- `0`：全部合格
- `1`：有规则不合格
- `2`：参数错误
- `3`：输入错误（路径/读取/解码等）
- `4`：配置错误
- `5`：内部错误

## 规则配置

参考：`examples/config.example.yaml`

支持环境变量展开：

- `${VAR}`
- `${VAR:-default}`

## 设计口径

- 目录默认递归
- 路径输出为绝对路径
- 自动识别文本/二进制
- 编码：UTF-8 优先，失败尝试 GBK/GB18030
- 字符数按 `rune`
- 最大行宽按显示宽度（含中日韩宽字符）
- tab 宽度按 4，按 tab stop 计算
- 默认启用 `.gitignore` 与内置忽略目录（`.git`/`node_modules`/`vendor`/`dist`/`build`）
