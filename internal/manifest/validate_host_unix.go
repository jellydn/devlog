//go:build !windows

package manifest

import (
	"fmt"
	"os"
	"syscall"
)

// ValidateHostPath checks that hostPath exists and is owned by the current user.
// This reduces the risk of rewriting manifests to point at an untrusted binary.
func ValidateHostPath(hostPath string) error {
	linkInfo, err := os.Lstat(hostPath)
	if err != nil {
		return fmt.Errorf("host path %q: %w", hostPath, err)
	}
	if err := checkHostOwnerAndMode(hostPath, linkInfo); err != nil {
		return err
	}
	info := linkInfo
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		info, err = os.Stat(hostPath)
		if err != nil {
			return fmt.Errorf("host path %q: %w", hostPath, err)
		}
		if err := checkHostOwnerAndMode(hostPath, info); err != nil {
			return err
		}
	}
	if info.IsDir() {
		return fmt.Errorf("host path %q is a directory", hostPath)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("host path %q is not a regular file", hostPath)
	}
	return nil
}

func checkHostOwnerAndMode(hostPath string, info os.FileInfo) error {
	if info.Mode().Perm()&0022 != 0 {
		return fmt.Errorf("host path %q is writable by group or others", hostPath)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("host path %q: cannot verify ownership", hostPath)
	}
	if int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("host path %q is not owned by the current user", hostPath)
	}
	return nil
}
