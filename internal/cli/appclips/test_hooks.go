package appclips

import (
	"github.com/dual1208/App-Store-Connect-CLI/internal/asc"
	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
)

// SetClientFactory replaces the ASC client factory for tests.
// It returns a restore function to reset the previous handler.
func SetClientFactory(fn func() (*asc.Client, error)) func() {
	previous := appClipsClientFactory
	if fn == nil {
		appClipsClientFactory = shared.GetASCClient
	} else {
		appClipsClientFactory = fn
	}
	return func() {
		appClipsClientFactory = previous
	}
}
