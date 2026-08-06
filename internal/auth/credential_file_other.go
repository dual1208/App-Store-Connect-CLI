//go:build !darwin && !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package auth

import "os"

func credentialFileHasMultipleLinks(_ *os.File, _ os.FileInfo) (bool, error) { return false, nil }
func credentialFileOwnedByCurrentUser(_ os.FileInfo) error                   { return nil }
