package web

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dual1208/App-Store-Connect-CLI/internal/privatefile"
)

const webPasswordStoreDirEnv = "ASC_WEB_PASSWORD_STORE_DIR"

// ErrPasswordStoreUnavailable indicates that the private file store cannot be used.
var ErrPasswordStoreUnavailable = errors.New("password file store unavailable")

// PasswordStoreBypassed remains for API compatibility. Password persistence is
// always file-backed and can be disabled by the caller's ordinary login flags.
func PasswordStoreBypassed() bool { return false }

func normalizedPasswordAppleID(appleID string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(appleID))
	if normalized == "" {
		return "", fmt.Errorf("apple id is required")
	}
	return normalized, nil
}

func webPasswordStoreDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(webPasswordStoreDirEnv)); custom != "" {
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".asc", "web-passwords"), nil
}

func passwordFilePath(appleID string) (string, error) {
	normalized, err := normalizedPasswordAppleID(appleID)
	if err != nil {
		return "", err
	}
	dir, err := webPasswordStoreDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(normalized))
	return filepath.Join(dir, hex.EncodeToString(sum[:])+".password"), nil
}

func LoadPassword(appleID string) (string, bool, error) {
	path, err := passwordFilePath(appleID)
	if err != nil {
		return "", false, err
	}
	if err := privatefile.EnsureDir(filepath.Dir(path)); err != nil {
		return "", false, fmt.Errorf("validate password store directory: %w", err)
	}
	data, err := privatefile.Read(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read password file: %w", err)
	}
	if len(data) == 0 {
		return "", false, nil
	}
	return string(data), true, nil
}

func PasswordStored(appleID string) (bool, error) {
	_, ok, err := LoadPassword(appleID)
	return ok, err
}

func StorePassword(appleID, password string) error {
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("password is required")
	}
	path, err := passwordFilePath(appleID)
	if err != nil {
		return err
	}
	if err := privatefile.WriteAtomically(path, []byte(password)); err != nil {
		return fmt.Errorf("store password file: %w", err)
	}
	return nil
}

func DeletePassword(appleID string) error {
	path, err := passwordFilePath(appleID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("delete password file: %w", err)
	}
	return nil
}

func DeleteAllPasswords() error {
	dir, err := webPasswordStoreDir()
	if err != nil {
		return err
	}
	if _, err := os.Lstat(dir); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err := privatefile.EnsureDir(dir); err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".password") {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
