package subscriptions

import (
	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/winbackoffers"
)

// SubscriptionsWinBackOffersCommand returns the canonical nested win-back offers tree.
func SubscriptionsWinBackOffersCommand() *ffcli.Command {
	return shared.RewriteCommandTreePath(
		winbackoffers.WinBackOffersCommand(),
		"asc win-back-offers",
		"asc subscriptions win-back-offers",
	)
}
