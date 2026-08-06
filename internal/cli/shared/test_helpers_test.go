package shared

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func captureOutput(t *testing.T, fn func()) (string, string) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = wOut, wErr
	outC, errC := make(chan string), make(chan string)
	go func() { var b bytes.Buffer; _, _ = io.Copy(&b, rOut); outC <- b.String() }()
	go func() { var b bytes.Buffer; _, _ = io.Copy(&b, rErr); errC <- b.String() }()
	fn()
	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	return <-outC, <-errC
}

func resetDefaultOutput(t *testing.T) {
	t.Helper()
	ResetDefaultOutputFormat()
	t.Cleanup(ResetDefaultOutputFormat)
}

func setTerminalDetection(t *testing.T, detector func(int) bool) {
	t.Helper()
	previous := isTerminal
	isTerminal = detector
	t.Cleanup(func() { isTerminal = previous })
}

func resetPrivateKeyTemp(t *testing.T) {
	t.Helper()
	CleanupTempPrivateKeys()
	t.Cleanup(CleanupTempPrivateKeys)
	t.Setenv("ASC_PRIVATE_KEY_PATH", "")
	t.Setenv("ASC_PRIVATE_KEY_B64", "")
	t.Setenv("ASC_PRIVATE_KEY", "")
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
}

func writeECDSAPEM(t *testing.T, path string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatal(err)
	}
}
