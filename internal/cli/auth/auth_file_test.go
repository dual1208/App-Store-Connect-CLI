package auth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	authsvc "github.com/dual1208/App-Store-Connect-CLI/internal/auth"
)

func TestLoginStorageMessageUsesNormalFile(t *testing.T) {
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	message, err := loginStorageMessage(false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(message, "private config file") {
		t.Fatalf("unexpected storage message: %q", message)
	}
}

func TestAuthLoginLocalWritesFileCredential(t *testing.T) {
	dir := t.TempDir()
	oldWorkingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWorkingDirectory) })

	keyPath := filepath.Join(dir, "AuthKey_TEST.p8")
	writeTestPrivateKey(t, keyPath)

	command := AuthLoginCommand()
	command.FlagSet.SetOutput(io.Discard)
	if err := command.Parse([]string{"--local", "--skip-validation", "--name", "demo", "--key-id", "KEY", "--issuer-id", "ISS", "--private-key", keyPath}); err != nil {
		t.Fatal(err)
	}
	if err := command.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	localPath := filepath.Join(dir, ".asc", "config.json")
	t.Setenv("ASC_CONFIG_PATH", localPath)
	credentials, err := authsvc.ListCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials) != 1 || credentials[0].Name != "demo" {
		t.Fatalf("unexpected credentials: %+v", credentials)
	}
}

func writeTestPrivateKey(t *testing.T, path string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
