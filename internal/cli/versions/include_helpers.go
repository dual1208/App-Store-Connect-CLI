package versions

import "github.com/dual1208/App-Store-Connect-CLI/internal/cli/shared"

func normalizeAppStoreVersionInclude(value string) ([]string, error) {
	return shared.NormalizeSelection(value, appStoreVersionIncludeList(), "--include")
}

func appStoreVersionIncludeList() []string {
	return []string{
		"ageRatingDeclaration",
		"appStoreReviewDetail",
		"appClipDefaultExperience",
		"appStoreVersionExperiments",
		"appStoreVersionExperimentsV2",
		"appStoreVersionSubmission",
		"customerReviews",
		"routingAppCoverage",
		"alternativeDistributionPackage",
		"gameCenterAppVersion",
	}
}
