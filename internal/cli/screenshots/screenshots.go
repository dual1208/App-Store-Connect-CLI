package screenshots

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/assets"
	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
)

// ScreenshotsCommand returns the top-level screenshots command.
func ScreenshotsCommand() *ffcli.Command {
	fs := flag.NewFlagSet("screenshots", flag.ExitOnError)

	return &ffcli.Command{
		Name:       "screenshots",
		ShortUsage: "asc screenshots <subcommand> [flags]",
		ShortHelp:  "Upload and manage App Store screenshots.",
		LongHelp: `Manage App Store screenshots through App Store Connect.

  asc screenshots list --version-localization "VERSION_LOCALIZATION_ID"
  asc screenshots sizes
  asc screenshots sizes --all
  asc screenshots validate --path "./screenshots/iphone" --device-type "IPHONE_65"
  asc screenshots upload --version-localization "VERSION_LOCALIZATION_ID" --path "./screenshots/iphone" --device-type "IPHONE_65"
  asc screenshots upload --app "123456789" --version "1.2.3" --path "./screenshots" --device-type "IPHONE_65"
  asc screenshots upload --version-localization "VERSION_LOCALIZATION_ID" --path "./screenshots/ipad" --device-type "IPAD_PRO_3GEN_129"
  asc screenshots download --version-localization "VERSION_LOCALIZATION_ID" --output-dir "./screenshots/downloaded"
  asc screenshots delete --id "SCREENSHOT_ID" --confirm

For most iOS submissions, one iPhone set (IPHONE_65) and one iPad set
(IPAD_PRO_3GEN_129) are enough. "asc screenshots sizes" focuses on these by
default; use --all only when you need the full matrix.`,
		FlagSet:   fs,
		UsageFunc: shared.DefaultUsageFunc,
		Subcommands: []*ffcli.Command{
			assets.AssetsScreenshotsListCommand(),
			assets.AssetsScreenshotsSizesCommand(),
			assets.AssetsScreenshotsValidateCommand(),
			assets.AssetsScreenshotsUploadCommand(),
			assets.AssetsScreenshotsDownloadCommand(),
			assets.AssetsScreenshotsDeleteCommand(),
		},
		Exec: func(ctx context.Context, args []string) error {
			return flag.ErrHelp
		},
	}
}
