//go:build windows

package manifest

import (
	"fmt"
	"os"
)

// ValidateHostPath checks that hostPath exists and is not a directory.
// Windows os.FileMode does not report ACLs, so this build does not reject a
// file that other users can rewrite. Unix builds check owner and mode.
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
