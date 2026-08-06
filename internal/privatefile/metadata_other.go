//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package privatefile

import "os"

func ownedByCurrentUser(os.FileInfo) error { return nil }
func hasExactlyOneLink(os.FileInfo) error  { return nil }
