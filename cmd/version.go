package cmd

import "fmt"

var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

func versionText() string {
	return fmt.Sprintf("syl-wordcount 版本：%s（commit: %s，构建时间: %s）", Version, Commit, BuildTime)
}
