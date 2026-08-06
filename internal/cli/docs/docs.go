package docs

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
)

// DocsCommand returns the docs command group.
func DocsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("docs", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "docs",
		ShortUsage: "asc docs <subcommand> [flags]",
		ShortHelp:  "Access embedded App Store Connect documentation guides.",
		LongHelp: `Access embedded App Store Connect documentation guides.

Examples:
  asc docs list
  asc docs show api-notes
  asc docs show workflows`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			DocsListCommand(),
			DocsShowCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return flag.ErrHelp
			}
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", args[0])
			return flag.ErrHelp
		},
	}
}
