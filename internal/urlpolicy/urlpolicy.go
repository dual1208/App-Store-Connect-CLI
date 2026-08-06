package urlpolicy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dual1208/App-Store-Connect-CLI/internal/urlsanitize"
)

var appleHostSuffixes = []string{
	"apple.com",
	"cdn-apple.com",
	"icloud-content.com",
	"mzstatic.com",
}

var signedCDNHostSuffixes = []string{
	"amazonaws.com",
	"azureedge.net",
	"cloudfront.net",
}

// ValidateAppleOrSignedHTTPS accepts only absolute HTTPS URLs without userinfo
// or fragments. Apple-owned hosts are accepted directly; known external CDN
// hosts must carry a recognized non-empty signature parameter.
func ValidateAppleOrSignedHTTPS(rawURL, purpose string) (*url.URL, error) {
	parsed, err := ParseHTTPS(rawURL, purpose)
	if err != nil {
		return nil, err
	}
	host := strings.ToLower(parsed.Hostname())
	if hostMatchesAny(host, appleHostSuffixes) {
		return parsed, nil
	}
	if hostMatchesAny(host, signedCDNHostSuffixes) && urlsanitize.HasSignedQuery(parsed.Query(), urlsanitize.DefaultSignedQueryKeys) {
		return parsed, nil
	}
	return nil, fmt.Errorf("%s host %q is not an allowed Apple or signed CDN host", purpose, parsed.Host)
}

// ParseHTTPS validates the common transport boundary for capability-bearing
// upload and download URLs.
func ParseHTTPS(rawURL, purpose string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("%s is required", purpose)
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid %s", purpose)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return nil, fmt.Errorf("%s must use https", purpose)
	}
	if parsed.Opaque != "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("%s must be an absolute URL with a host", purpose)
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("%s must not contain userinfo", purpose)
	}
	if parsed.Fragment != "" {
		return nil, fmt.Errorf("%s must not contain a fragment", purpose)
	}
	return parsed, nil
}

// ClientWithoutRedirects clones an HTTP client and rejects every redirect.
func ClientWithoutRedirects(client *http.Client) *http.Client {
	if client == nil {
		client = &http.Client{}
	}
	safeClient := *client
	safeClient.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &safeClient
}

func hostMatchesAny(host string, suffixes []string) bool {
	host = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
	for _, suffix := range suffixes {
		if host == suffix || strings.HasSuffix(host, "."+suffix) {
			return true
		}
	}
	return false
}
