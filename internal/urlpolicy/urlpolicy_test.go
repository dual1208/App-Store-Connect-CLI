package urlpolicy

import (
	"net/http"
	"testing"
)

func TestValidateAppleOrSignedHTTPS(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "Apple", url: "https://iosapps-ssl.itunes.apple.com/upload"},
		{name: "Apple CDN", url: "https://is1-ssl.mzstatic.com/image.png"},
		{name: "signed CDN", url: "https://bucket.s3.amazonaws.com/file?X-Amz-Signature=abc"},
		{name: "HTTP", url: "http://iosapps-ssl.itunes.apple.com/upload", wantErr: true},
		{name: "userinfo", url: "https://user:secret@apple.com/file", wantErr: true},
		{name: "fragment", url: "https://apple.com/file#secret", wantErr: true},
		{name: "untrusted", url: "https://example.com/file", wantErr: true},
		{name: "unsigned CDN", url: "https://bucket.s3.amazonaws.com/file", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := ValidateAppleOrSignedHTTPS(test.url, "asset URL")
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateAppleOrSignedHTTPS() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestClientWithoutRedirectsDoesNotMutateInput(t *testing.T) {
	original := &http.Client{}
	safe := ClientWithoutRedirects(original)
	if original.CheckRedirect != nil {
		t.Fatal("input client was mutated")
	}
	if safe.CheckRedirect == nil {
		t.Fatal("safe client does not reject redirects")
	}
}
