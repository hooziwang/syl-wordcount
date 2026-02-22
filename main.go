package main

import (
	"os"

	"syl-wordcount/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
