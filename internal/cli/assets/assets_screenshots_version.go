package assets

import (
	"context"
	"strings"

	"github.com/dual1208/App-Store-Connect-CLI/internal/asc"
	"github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"
)

func resolveScreenshotPlanVersion(ctx context.Context, client *asc.Client, appID, version, versionID, platform string) (string, string, string, error) {
	if strings.TrimSpace(versionID) != "" {
		versionData, err := shared.ResolveOwnedAppStoreVersionByID(ctx, client, appID, versionID, platform)
		if err != nil {
			return "", "", "", err
		}
		resolvedPlatform := strings.TrimSpace(string(versionData.Attributes.Platform))
		return strings.TrimSpace(versionData.ID), strings.TrimSpace(versionData.Attributes.VersionString), resolvedPlatform, nil
	}

	resolvedVersionID, err := shared.ResolveAppStoreVersionID(ctx, client, appID, strings.TrimSpace(version), strings.TrimSpace(platform))
	if err != nil {
		return "", "", "", err
	}
	return resolvedVersionID, strings.TrimSpace(version), strings.TrimSpace(platform), nil
}
