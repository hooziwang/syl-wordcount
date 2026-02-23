package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"syl-wordcount/internal/app"
	"syl-wordcount/internal/output"
	"syl-wordcount/internal/scan"
)

type commonFlags struct {
	Config      string
	Format      string
	Jobs        int
	MaxFileSize string
	CheckAll    bool
	ShowVersion bool
}

func Execute() int {
	root := NewRootCmd(os.Stdout, os.Stderr)
	root.SetArgs(normalizeArgs(os.Args[1:]))
	if err := root.Execute(); err != nil {
		var ee *ExitError
		if errors.As(err, &ee) {
			if ee.Msg != "" {
				fmt.Fprintln(os.Stderr, ee.Msg)
			}
			return ee.Code
		}
		code := ExitInternal
		errCode := "runtime_failed"
		if strings.Contains(err.Error(), "unknown command") || strings.Contains(err.Error(), "unknown flag") {
			code = ExitArg
			errCode = "unknown_command"
		}
		writeCLIError(os.Stdout, detectFormatFromArgs(os.Args[1:]), "unknown", os.Args[1:], errCode, "arg", "", err.Error(), code)
		fmt.Fprintln(os.Stderr, err.Error())
		return code
	}
	return ExitOK
}

func NewRootCmd(stdout, stderr io.Writer) *cobra.Command {
	flags := &commonFlags{}
	root := &cobra.Command{
		Use:           "syl-wordcount [paths...]",
		Short:         "统计文本文件字数/行数/最大行宽，并支持规则校验",
		Long:          rootLongHelp(),
		Example:       rootExampleHelp(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.ShowVersion {
				printVersion(stdout)
				return nil
			}
			if len(args) == 0 {
				_ = cmd.Help()
				return &ExitError{Code: ExitArg, Msg: "还没传输入路径，至少要给一个文件或目录。示例：syl-wordcount /path/to/input_dir"}
			}
			return runMode(stdout, flags, app.ModeStats, args)
		},
	}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.CompletionOptions.HiddenDefaultCmd = true
	bindCommon(root, flags)

	internalStatsCmd := &cobra.Command{
		Use:           "__stats [paths...]",
		Short:         "internal stats entry",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMode(stdout, flags, app.ModeStats, args)
		},
	}
	root.AddCommand(internalStatsCmd)

	checkCmd := &cobra.Command{
		Use:           "check [paths...]",
		Short:         "按规则检查文本质量并输出违规定位",
		Long:          checkLongHelp(),
		Example:       checkExampleHelp(),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if flags.ShowVersion {
				printVersion(stdout)
				return nil
			}
			return runMode(stdout, flags, app.ModeCheck, args)
		},
	}
	checkCmd.Flags().BoolVar(&flags.CheckAll, "all", false, "输出全量结果（包含 pass 事件）")
	root.AddCommand(checkCmd)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion(stdout)
		},
	}
	root.AddCommand(versionCmd)
	return root
}

func bindCommon(cmd *cobra.Command, flags *commonFlags) {
	cmd.PersistentFlags().StringVar(&flags.Config, "config", "", "YAML 规则配置文件路径（check 模式可选，未传则尝试读取环境变量规则）")
	cmd.PersistentFlags().StringVar(&flags.Format, "format", "ndjson", "输出格式：ndjson/json")
	cmd.PersistentFlags().IntVar(&flags.Jobs, "jobs", app.DefaultJobs(), "并发任务数（默认 min(8, CPU核数)）")
	cmd.PersistentFlags().StringVar(&flags.MaxFileSize, "max-file-size", "10MB", "单文件最大处理大小，超出则跳过（如 10MB）")
	cmd.PersistentFlags().BoolVarP(&flags.ShowVersion, "version", "v", false, "显示版本信息")
}

func runMode(stdout io.Writer, flags *commonFlags, mode app.Mode, args []string) error {
	if len(args) == 0 {
		msg := "还没传输入路径，至少要给一个文件或目录。示例：syl-wordcount /path/to/input_dir"
		writeCLIError(stdout, flags.Format, string(mode), args, "arg_missing_paths", "arg", "", msg, ExitArg)
		return &ExitError{Code: ExitArg, Msg: msg}
	}
	if err := scan.ValidateFormat(flags.Format); err != nil {
		writeCLIError(stdout, flags.Format, string(mode), args, "invalid_output_format", "arg", "", err.Error(), ExitArg)
		return &ExitError{Code: ExitArg, Msg: err.Error()}
	}
	maxBytes, err := parseSize(flags.MaxFileSize)
	if err != nil {
		writeCLIError(stdout, flags.Format, string(mode), args, "invalid_max_file_size", "arg", "", err.Error(), ExitArg)
		return &ExitError{Code: ExitArg, Msg: err.Error()}
	}
	cwd, err := os.Getwd()
	if err != nil {
		msg := "读取当前目录失败"
		writeCLIError(stdout, flags.Format, string(mode), args, "cwd_failed", "internal", "", msg, ExitInternal)
		return &ExitError{Code: ExitInternal, Msg: msg}
	}
	paths := app.NormalizePaths(args, cwd)
	if len(paths) == 0 {
		msg := "输入路径为空或无效"
		writeCLIError(stdout, flags.Format, string(mode), args, "invalid_input_paths", "arg", "", msg, ExitArg)
		return &ExitError{Code: ExitArg, Msg: msg}
	}
	res, err := app.Run(app.Options{
		Mode:             mode,
		Paths:            paths,
		CWD:              cwd,
		ConfigPath:       flags.Config,
		Format:           flags.Format,
		Jobs:             flags.Jobs,
		MaxFileSizeBytes: maxBytes,
		Version:          Version,
		Args:             os.Args[1:],
	})
	if err != nil {
		switch err.(type) {
		case *app.ArgErr:
			writeCLIError(stdout, flags.Format, string(mode), args, "arg_invalid", "arg", "", err.Error(), ExitArg)
			return &ExitError{Code: ExitArg, Msg: err.Error()}
		case *app.ConfigErr:
			code := "check_config_invalid"
			if strings.Contains(err.Error(), "check 模式需要规则") {
				code = "check_rules_missing"
			}
			writeCLIError(stdout, flags.Format, string(mode), args, code, "config", "", err.Error(), ExitConfig)
			return &ExitError{Code: ExitConfig, Msg: err.Error()}
		default:
			writeCLIError(stdout, flags.Format, string(mode), args, "runtime_failed", "internal", "", err.Error(), ExitInternal)
			return &ExitError{Code: ExitInternal, Msg: err.Error()}
		}
	}
	events := eventsForOutput(mode, flags.CheckAll, res.Events)
	if werr := output.Write(stdout, flags.Format, events); werr != nil {
		msg := fmt.Sprintf("输出结果失败：%v", werr)
		return &ExitError{Code: ExitInternal, Msg: msg}
	}
	code := 0
	if res.HasInternalErr {
		code = ExitInternal
	} else if res.HasConfigErr {
		code = ExitConfig
	} else if res.HasInputErr {
		code = ExitInput
	} else if res.HasViolation {
		code = ExitViolation
	}
	if code != 0 {
		return &ExitError{Code: code}
	}
	return nil
}

func eventsForOutput(mode app.Mode, checkAll bool, events []map[string]any) []map[string]any {
	if mode != app.ModeCheck || checkAll {
		return events
	}
	filtered := make([]map[string]any, 0, len(events))
	for _, e := range events {
		t, _ := e["type"].(string)
		if t == "pass" {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

func parseSize(s string) (int64, error) {
	v := strings.TrimSpace(strings.ToUpper(s))
	if v == "" {
		return 10 * 1024 * 1024, nil
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
	for _, u := range units {
		if strings.HasSuffix(v, u.U) {
			n := strings.TrimSpace(strings.TrimSuffix(v, u.U))
			f, err := strconv.ParseFloat(n, 64)
			if err != nil {
				return 0, fmt.Errorf("--max-file-size 参数无效：%s", s)
			}
			return int64(f * float64(u.M)), nil
		}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("--max-file-size 参数无效：%s", s)
	}
	return n, nil
}

func normalizeArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	first := args[0]
	switch first {
	case "stats", "check", "version", "help", "completion", "__stats":
		return args
	}
	if strings.HasPrefix(first, "-") {
		return args
	}
	return append([]string{"__stats"}, args...)
}
