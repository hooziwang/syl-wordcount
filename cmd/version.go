package cmd

import (
	"fmt"
	"io"

	"github.com/hooziwang/daddylovesyl"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func versionText() string {
	return fmt.Sprintf("syl-wordcount 版本：%s（commit: %s，构建时间: %s）", Version, Commit, BuildTime)
}

func printVersion(w io.Writer) {
	fmt.Fprintln(w, versionText())
	fmt.Fprintln(w, daddylovesyl.Render(w))
}
