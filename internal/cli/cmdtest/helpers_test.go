package cmdtest

import (
	"testing"

	"github.com/peterbourgon/ff/v3/ffcli"

	cmd "github.com/dual1208/App-Store-Connect-CLI/cmd"
	"github.com/dual1208/App-Store-Connect-CLI/internal/asc"
	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
)

func resetCmdtestState() {
	asc.ResetConfigCacheForTest()
	shared.ResetDefaultOutputFormat()
	shared.ResetTierCacheForTest()
}

func setCmdtestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func RootCommand(version string) *ffcli.Command {
	resetCmdtestState()
	return cmd.RootCommand(version)
}

func findSubcommand(root *ffcli.Command, path ...string) *ffcli.Command {
	current := root
	for _, part := range path {
		var next *ffcli.Command
		for _, subcommand := range current.Subcommands {
			if subcommand.Name == part {
				next = subcommand
				break
			}
		}
		if next == nil {
			return nil
		}
		current = next
	}
	return current
}

type ReportedError = shared.ReportedError
