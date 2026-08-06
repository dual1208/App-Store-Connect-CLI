package main

import (
	"fmt"
	"os"

	"github.com/dual1208/App-Store-Connect-CLI/cmd"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func versionInfoString() string {
	return fmt.Sprintf("%s (commit: %s, date: %s)", version, commit, date)
}

func run(args []string) int {
	return cmd.Run(args, versionInfoString())
}

func main() {
	os.Exit(run(os.Args[1:]))
}
