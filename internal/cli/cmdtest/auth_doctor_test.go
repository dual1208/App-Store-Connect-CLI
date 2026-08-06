package cmdtest

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthDoctorTextRedactsCredentialIdentifiers(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("ASC_KEY_ID", "ABC123SECRET")
	t.Setenv("ASC_ISSUER_ID", "issuer-uuid")
	t.Setenv("ASC_PRIVATE_KEY_PATH", "/tmp/AuthKey.p8")

	root := RootCommand("1.2.3")
	root.FlagSet.SetOutput(io.Discard)
	stdout, _ := captureOutput(t, func() {
		if err := root.Parse([]string{"auth", "doctor"}); err != nil {
			t.Fatalf("parse error: %v", err)
		}
		if err := root.Run(context.Background()); err != nil {
			t.Fatalf("run error: %v", err)
		}
	})

	if !strings.Contains(stdout, "ASC_KEY_ID is set") || !strings.Contains(stdout, "ASC_ISSUER_ID is set") {
		t.Fatalf("expected credential presence messages, got %q", stdout)
	}
	if strings.Contains(stdout, "ABC123SECRET") || strings.Contains(stdout, "issuer-uuid") {
		t.Fatalf("expected credential identifiers to be redacted, got %q", stdout)
	}
}
