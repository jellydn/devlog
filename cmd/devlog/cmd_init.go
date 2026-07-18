package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jellydn/devlog/internal/config"
)

func cmdInit(cfg *config.Config, args []string) error {
	// Check if devlog.yml already exists
	configPath := filepath.Join(".", "devlog.yml")
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("devlog.yml already exists in current directory")
	}

	// Get current directory name for defaults
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	projectName := filepath.Base(cwd)

	// Detect if this is a monorepo by checking for common patterns
	isMonorepo := false
	monorepoIndicators := []string{"packages", "apps", "services"}
	for _, dir := range monorepoIndicators {
		if _, err := os.Stat(dir); err == nil {
			isMonorepo = true
			break
		}
	}

	// Create template content
	template := generateTemplate(projectName, isMonorepo)

	// Write the file
	if err := os.WriteFile(configPath, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write devlog.yml: %w", err)
	}

	fmt.Printf("Created devlog.yml in current directory\n")
	fmt.Printf("Project: %s\n", projectName)
	if isMonorepo {
		fmt.Printf("Detected monorepo structure\n")
	}
	fmt.Printf("\nEdit devlog.yml to customize your configuration, then run:\n")
	fmt.Printf("  devlog up\n")

	return nil
}

// generateTemplate creates the YAML template content
func generateTemplate(projectName string, isMonorepo bool) string {
	if isMonorepo {
		return fmt.Sprintf(`version: "1.0"
project: %s
logs_dir: ./logs
run_mode: timestamped # timestamped | overwrite

tmux:
  session: %s
  windows:
    - name: dev
      panes:
        - cmd: npm run dev
          log: server/web.log
        - cmd: pnpm --filter api dev
          log: server/api.log

browser:
  # Matches any port and path on localhost (e.g., http://localhost:3000/app)
  urls:
    - "http://localhost:*/*"
  file: browser/console.log
  levels:
    - error
    - warn
    - info
    - log
`, projectName, projectName)
	}

	return fmt.Sprintf(`version: "1.0"
project: %s
logs_dir: ./logs
run_mode: timestamped # timestamped | overwrite

tmux:
  session: %s
  windows:
    - name: dev
      panes:
        - cmd: npm run dev
          log: server.log

browser:
  # Matches any port and path on localhost (e.g., http://localhost:3000/app)
  urls:
    - "http://localhost:*/*"
  file: browser.log
  levels:
    - error
    - warn
    - info
    - log
`, projectName, projectName)
}
