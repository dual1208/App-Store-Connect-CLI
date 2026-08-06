package shared

import "github.com/dual1208/App-Store-Connect-CLI/internal/asc"

// ResolveAppStoreVersionState prefers the app version state when available.
func ResolveAppStoreVersionState(attrs asc.AppStoreVersionAttributes) string {
	if attrs.AppVersionState != "" {
		return attrs.AppVersionState
	}
	return attrs.AppStoreState
}
