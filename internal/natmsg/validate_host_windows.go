//go:build windows

package natmsg

import (
	"fmt"
	"os"
)

// ValidateHostPath checks that hostPath exists (ownership checks are Unix-only).
func ValidateHostPath(hostPath string) error {
	info, err := os.Stat(hostPath)
	if err != nil {
		return fmt.Errorf("host path %q: %w", hostPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("host path %q is a directory", hostPath)
	}
	return nil
}
