package privatefile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadPrivateFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	path := filepath.Join(dir, "secret")
	if err := WriteAtomically(path, []byte("value")); err != nil {
		t.Fatal(err)
	}
	data, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "value" {
		t.Fatalf("Read() = %q", data)
	}
	dirInfo, _ := os.Stat(dir)
	fileInfo, _ := os.Stat(path)
	if dirInfo.Mode().Perm() != 0o700 || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected modes: dir=%#o file=%#o", dirInfo.Mode().Perm(), fileInfo.Mode().Perm())
	}
}

func TestReadRejectsSymlinkAndHardLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("value"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	if _, err := Read(link); err == nil {
		t.Fatal("Read(symlink) expected error")
	}
	hard := filepath.Join(dir, "hard")
	if err := os.Link(target, hard); err != nil {
		t.Skip(err)
	}
	if _, err := Read(target); err == nil {
		t.Fatal("Read(multiply-linked file) expected error")
	}
}

func TestReadRejectsPermissiveFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("value"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("Read(permissive file) expected error")
	}
}
