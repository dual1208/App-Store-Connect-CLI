package web

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestPasswordStoreUsesPrivateFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "passwords")
	t.Setenv(webPasswordStoreDirEnv, dir)
	if err := StorePassword("User@Example.com", "secret"); err != nil {
		t.Fatal(err)
	}
	password, ok, err := LoadPassword("user@example.com")
	if err != nil || !ok || password != "secret" {
		t.Fatalf("LoadPassword() = %q, %v, %v", password, ok, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("password files = %d", len(entries))
	}
	info, _ := entries[0].Info()
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("password mode = %#o", info.Mode().Perm())
	}
}

func TestSessionStoreUsesPrivateFiles(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	t.Setenv(webSessionCacheDirEnv, dir)
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	base, _ := url.Parse("https://appstoreconnect.apple.com/")
	jar.SetCookies(base, []*http.Cookie{{Name: "session", Value: "value", Secure: true}})
	if err := PersistSession(&AuthSession{Client: &http.Client{Jar: jar}, UserEmail: "user@example.com"}); err != nil {
		t.Fatal(err)
	}
	loaded, ok, err := LoadCachedSession("user@example.com")
	if err != nil || !ok || loaded == nil {
		t.Fatalf("LoadCachedSession() = %v, %v, %v", loaded, ok, err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("session files = %d", len(entries))
	}
}
