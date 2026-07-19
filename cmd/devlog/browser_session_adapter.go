package main

import (
	"github.com/jellydn/devlog/internal/manifest"
	"github.com/jellydn/devlog/internal/tmux"
)

// manifestAdapter adapts package-level manifest functions to browsersession.ManifestOps.
type manifestAdapter struct{}

func (manifestAdapter) FindDevlogHostBinary() (string, error) {
	return manifest.FindDevlogHostBinary()
}
func (manifestAdapter) ValidateHostPath(path string) error {
	return manifest.ValidateHostPath(path)
}
func (manifestAdapter) RepairStaleManifestPaths(hostPath string) (int, error) {
	return manifest.RepairStaleManifestPaths(hostPath)
}
func (manifestAdapter) UpdateManifestPath(newPath string) error {
	return manifest.UpdateManifestPath(newPath)
}
func (manifestAdapter) ReadManifestPaths() (map[string]string, error) {
	return manifest.ReadManifestPaths()
}
func (manifestAdapter) IsManifestPathInUse(targetPath string) (bool, error) {
	return manifest.IsManifestPathInUse(targetPath)
}
func (manifestAdapter) GetChromeNativeMessagingDir() string {
	return manifest.GetChromeNativeMessagingDir()
}
func (manifestAdapter) GetBraveNativeMessagingDir() string {
	return manifest.GetBraveNativeMessagingDir()
}
func (manifestAdapter) GetFirefoxNativeMessagingDirs() []string {
	return manifest.GetFirefoxNativeMessagingDirs()
}

// tmuxAdapter adapts tmux.Runner to browsersession.SessionChecker.
// SessionExists ignores the name argument and checks the runner's configured session,
// matching the previous helpers.go behavior of tmux.NewRunner(otherSession).SessionExists().
type tmuxSessionChecker struct{}

func (tmuxSessionChecker) SessionExists(name string) bool {
	return tmux.NewRunner(name).SessionExists()
}
