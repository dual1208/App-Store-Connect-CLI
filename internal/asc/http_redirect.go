package asc

import (
	"net/http"

	"github.com/dual1208/App-Store-Connect-CLI/internal/urlpolicy"
)

func clientWithoutRedirects(client *http.Client) *http.Client {
	if client == nil {
		client = newDefaultHTTPClient(ResolveTimeout())
	}
	return urlpolicy.ClientWithoutRedirects(client)
}
