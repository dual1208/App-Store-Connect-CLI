package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

var testConfigPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "asc-auth-file-tests-")
	if err != nil {
		panic(err)
	}
	testConfigPath = filepath.Join(dir, "config.json")
	_ = os.Setenv("ASC_CONFIG_PATH", testConfigPath)
	_ = os.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func writeECDSAPEM(t *testing.T, path string, mode os.FileMode, _ bool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: data}), mode); err != nil {
		t.Fatal(err)
	}
}
