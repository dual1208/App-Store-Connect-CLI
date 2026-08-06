package privatefile

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/dual1208/App-Store-Connect-CLI/internal/secureopen"
)

// EnsureDir creates or validates a private directory owned by the current user.
func EnsureDir(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("private directory %q must be a real directory", path)
	}
	if err := ownedByCurrentUser(info); err != nil {
		return err
	}
	if info.Mode().Perm() != 0o700 {
		if err := os.Chmod(path, 0o700); err != nil {
			return fmt.Errorf("set private directory %q to mode 0700: %w", path, err)
		}
	}
	return nil
}

// Read reads a private normal file without following its final path component.
func Read(path string) ([]byte, error) {
	file, err := secureopen.OpenExistingNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err := validateFile(file); err != nil {
		return nil, err
	}
	return io.ReadAll(file)
}

// WriteAtomically writes a private file through a same-directory temporary file.
func WriteAtomically(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("private file %q must be a regular non-symlink file", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := secureopen.CreateTempNoFollow(dir, ".private-*", 0o600)
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	ok = true
	return nil
}

func validateFile(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("private file must be a regular file")
	}
	if info.Mode().Perm() != 0o600 {
		return fmt.Errorf("private file %q must have mode 0600", file.Name())
	}
	if err := ownedByCurrentUser(info); err != nil {
		return err
	}
	if err := hasExactlyOneLink(info); err != nil {
		return err
	}
	return nil
}
