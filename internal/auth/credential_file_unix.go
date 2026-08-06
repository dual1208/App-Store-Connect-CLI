//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package auth

import (
	"fmt"
	"os"
	"syscall"
)

func credentialFileHasMultipleLinks(_ *os.File, info os.FileInfo) (bool, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink != 1, nil
}

func credentialFileOwnedByCurrentUser(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("failed to inspect private key ownership")
	}
	if int(stat.Uid) != os.Geteuid() {
		return fmt.Errorf("private key file must be owned by the current user")
	}
	return nil
}
