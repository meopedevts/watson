package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the watson daemon.
type Config struct {
	// GitHubReviewerUsername is the GitHub username whose review requests are polled.
	// Required — loaded from GITHUB_REVIEWER_USERNAME.
	GitHubReviewerUsername string

	// PollIntervalMinutes controls how often GitHub is polled.
	// Loaded from POLL_INTERVAL_MINUTES; default: 15.
	PollIntervalMinutes int

	// ClaudeModel is the model identifier passed to the claude CLI.
	// Loaded from CLAUDE_MODEL; default: "claude-sonnet-4-20250514".
	ClaudeModel string

	// RepoBaseDir is the parent directory for temporary clone directories.
	// Loaded from REPO_BASE_DIR; default: "/tmp/watson".
	RepoBaseDir string

	// GitSSHHost is an optional SSH host alias defined in ~/.ssh/config.
	// When set, clones use SSH with the alias: git@<GitSSHHost>:<owner>/<repo>.git
	// When empty, clones use the standard HTTPS URL: https://github.com/<owner>/<repo>
	// Loaded from GIT_SSH_HOST.
	GitSSHHost string

	// DryRun is set by the --dry-run CLI flag (not an env var).
	// When true, reviews are printed to stdout instead of posted on GitHub.
	DryRun bool
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is missing or malformed.
func Load() (*Config, error) {
	username := os.Getenv("GITHUB_REVIEWER_USERNAME")
	if username == "" {
		return nil, fmt.Errorf("GITHUB_REVIEWER_USERNAME is required")
	}

	pollInterval := 15
	if raw := os.Getenv("POLL_INTERVAL_MINUTES"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("POLL_INTERVAL_MINUTES must be a positive integer, got %q", raw)
		}
		pollInterval = v
	}

	model := getenv("CLAUDE_MODEL", "claude-sonnet-4-20250514")
	baseDir := getenv("REPO_BASE_DIR", "/tmp/watson")
	sshHost := os.Getenv("GIT_SSH_HOST")

	return &Config{
		GitHubReviewerUsername: username,
		PollIntervalMinutes:    pollInterval,
		ClaudeModel:            model,
		RepoBaseDir:            baseDir,
		GitSSHHost:             sshHost,
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
