package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorConfigPermissionsWarning(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	configPath := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(configPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write config error: %v", err)
	}
	t.Setenv("ASC_CONFIG_PATH", configPath)

	report := Doctor(DoctorOptions{})
	section := findDoctorSection(t, report, "Storage")
	if !sectionHasStatus(section, DoctorWarn, "Config file permissions") {
		t.Fatalf("expected config permissions warning, got %#v", section.Checks)
	}

	Doctor(DoctorOptions{Fix: true})
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config error: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected config permissions fixed to 0600, got %#o", info.Mode().Perm())
	}
}

func TestDoctorEnvironmentRedactsCredentialIdentifiers(t *testing.T) {
	t.Setenv("ASC_BYPASS_KEYCHAIN", "1")
	t.Setenv("ASC_CONFIG_PATH", filepath.Join(t.TempDir(), "config.json"))
	t.Setenv("ASC_KEY_ID", "ABC123SECRET")
	t.Setenv("ASC_ISSUER_ID", "issuer-uuid")
	t.Setenv("ASC_PRIVATE_KEY_PATH", "/tmp/AuthKey.p8")

	report := Doctor(DoctorOptions{})
	section := findDoctorSection(t, report, "Environment")
	if !sectionHasStatus(section, DoctorInfo, "ASC_KEY_ID is set") || !sectionHasStatus(section, DoctorInfo, "ASC_ISSUER_ID is set") {
		t.Fatalf("expected credential presence messages, got %#v", section.Checks)
	}
	for _, check := range section.Checks {
		if strings.Contains(check.Message, "ABC123SECRET") || strings.Contains(check.Message, "issuer-uuid") {
			t.Fatalf("credential identifier leaked in message: %q", check.Message)
		}
	}
}

func findDoctorSection(t *testing.T, report DoctorReport, title string) DoctorSection {
	t.Helper()
	for _, section := range report.Sections {
		if section.Title == title {
			return section
		}
	}
	t.Fatalf("expected section %q, got %#v", title, report.Sections)
	return DoctorSection{}
}

func sectionHasStatus(section DoctorSection, status DoctorStatus, contains string) bool {
	for _, check := range section.Checks {
		if check.Status == status && strings.Contains(check.Message, contains) {
			return true
		}
	}
	return false
}
