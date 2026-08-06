//go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly

package privatefile

import (
	"fmt"
	"os"
	"syscall"
)

func ownedByCurrentUser(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot determine owner of %q", info.Name())
	}
	if stat.Uid != uint32(os.Getuid()) {
		return fmt.Errorf("%q must be owned by the current user", info.Name())
	}
	return nil
}

func hasExactlyOneLink(info os.FileInfo) error {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot determine link count of %q", info.Name())
	}
	if stat.Nlink != 1 {
		return fmt.Errorf("%q must have exactly one hard link", info.Name())
	}
	return nil
}
