//go:build !windows

package natmsg

import (
	"fmt"
	"os"
	"syscall"
)

// ValidateHostPath checks that hostPath exists and is owned by the current user.
// This reduces the risk of rewriting manifests to point at an untrusted binary.
func ValidateHostPath(hostPath string) error {
	info, err := os.Stat(hostPath)
	if err != nil {
		return fmt.Errorf("host path %q: %w", hostPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("host path %q is a directory", hostPath)
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}
	if int(stat.Uid) != os.Getuid() {
		return fmt.Errorf("host path %q is not owned by the current user", hostPath)
	}
	return nil
}
