package fileonlytls

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCustomCASucceeds(t *testing.T) {
	server, roots := newTLSServer(t, "127.0.0.1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	client := newClient(roots, "")

	response, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("request with custom CA failed: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected response status: %s", response.Status)
	}
}

func TestHostnameMismatchFails(t *testing.T) {
	server, roots := newTLSServer(t, "127.0.0.1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	client := newClient(roots, "wrong.example")

	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("TLS request with a mismatched hostname unexpectedly succeeded")
	}
	var hostnameError x509.HostnameError
	if !errors.As(err, &hostnameError) {
		t.Fatalf("expected hostname verification failure, got %T: %v", err, err)
	}
}

func TestUntrustedRootFails(t *testing.T) {
	server, _ := newTLSServer(t, "127.0.0.1", time.Now().Add(-time.Hour), time.Now().Add(time.Hour))
	client := newClient(x509.NewCertPool(), "")

	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("TLS request with an untrusted root unexpectedly succeeded")
	}
	var unknownAuthorityError x509.UnknownAuthorityError
	if !errors.As(err, &unknownAuthorityError) {
		t.Fatalf("expected unknown-authority failure, got %T: %v", err, err)
	}
}

func TestExpiredCertificateFails(t *testing.T) {
	server, roots := newTLSServer(t, "127.0.0.1", time.Now().Add(-2*time.Hour), time.Now().Add(-time.Hour))
	client := newClient(roots, "")

	_, err := client.Get(server.URL)
	if err == nil {
		t.Fatal("TLS request with an expired certificate unexpectedly succeeded")
	}
	var certificateInvalidError x509.CertificateInvalidError
	if !errors.As(err, &certificateInvalidError) {
		t.Fatalf("expected certificate-validity failure, got %T: %v", err, err)
	}
}

func TestASCReadOnlyLiveTLS(t *testing.T) {
	if os.Getenv("ASC_FILEONLY_LIVE_TLS") != "1" {
		t.Skip("set ASC_FILEONLY_LIVE_TLS=1 only after offline tests pass")
	}

	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.appstoreconnect.apple.com/v1/apps?limit=1", nil)
	if err != nil {
		t.Fatalf("create read-only App Store Connect request: %v", err)
	}
	client := &http.Client{
		Timeout: 20 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
		},
	}
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("read-only App Store Connect TLS request failed: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusUnauthorized && response.StatusCode != http.StatusForbidden {
		t.Fatalf("unexpected unauthenticated App Store Connect status: %s", response.Status)
	}
}

func newClient(roots *x509.CertPool, serverName string) *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: nil,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				RootCAs:    roots,
				ServerName: serverName,
			},
		},
	}
}

func newTLSServer(t *testing.T, host string, notBefore, notAfter time.Time) (*httptest.Server, *x509.CertPool) {
	t.Helper()

	rootKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate root key: %v", err)
	}
	rootTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "File-only test root"},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create root certificate: %v", err)
	}
	rootCertificate, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatalf("parse root certificate: %v", err)
	}

	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate server key: %v", err)
	}
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: host},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP(host)},
	}
	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, rootCertificate, &serverKey.PublicKey, rootKey)
	if err != nil {
		t.Fatalf("create server certificate: %v", err)
	}
	serverKeyDER := x509.MarshalPKCS1PrivateKey(serverKey)
	serverCertificate, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}),
		pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: serverKeyDER}),
	)
	if err != nil {
		t.Fatalf("create TLS key pair: %v", err)
	}

	server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.WriteHeader(http.StatusNoContent)
	}))
	server.TLS = &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{serverCertificate},
	}
	server.StartTLS()
	t.Cleanup(server.Close)

	roots := x509.NewCertPool()
	roots.AddCert(rootCertificate)
	return server, roots
}
