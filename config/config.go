package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// Config holds all runtime configuration for the watson daemon.
type Config struct {
	// GitHubReviewerUsername is the GitHub login of the authenticated user.
	// Resolved once on startup via "gh api user" and stored here for the
	// lifetime of the daemon. Not configurable via env var.
	GitHubReviewerUsername string

	// PollIntervalMinutes controls how often GitHub is polled.
	// Loaded from POLL_INTERVAL_MINUTES; default: 15.
	PollIntervalMinutes int

	// ClaudeModel is the model identifier passed to the claude CLI.
	// Loaded from CLAUDE_MODEL; default: "claude-sonnet-4-20250514".
	ClaudeModel string

	// RepoBaseDir is the parent directory for temporary clone directories.
	// Loaded from REPO_BASE_DIR; default: filepath.Join(os.TempDir(), "watson").
	RepoBaseDir string

	// GitSSHHost is an optional SSH host alias defined in ~/.ssh/config.
	// When set, clones use SSH with the alias: git@<GitSSHHost>:<owner>/<repo>.git
	// When empty, clones use the standard HTTPS URL: https://github.com/<owner>/<repo>
	// Loaded from GIT_SSH_HOST.
	GitSSHHost string

	// ReviewTTLHours controls how long a reviewed PR stays in the in-memory cache.
	// After this period, the entry is removed: Watson will no longer check for
	// @mentions on that PR, and if it is still open and review is requested again,
	// it will be treated as a new PR.
	//
	// This value directly bounds memory usage and the number of GitHub API calls
	// made per tick for mention checking (one call per cached PR per tick).
	// Lower values reduce API usage; higher values allow Watson to respond to
	// mentions on older PRs.
	//
	// Default: 168 (7 days). Configurable via REVIEW_TTL_HOURS.
	ReviewTTLHours int

	// ReReviewCooldownMinutes is the minimum time that must elapse between two
	// consecutive reviews of the same PR (first-time or re-review). Mentions that
	// arrive within this window are ignored until the cooldown expires.
	//
	// This prevents comment spam when a PR receives multiple @mentions in rapid
	// succession. Set to a value >= POLL_INTERVAL_MINUTES to ensure at most one
	// re-review per mention burst.
	//
	// Default: 60 minutes. Configurable via RE_REVIEW_COOLDOWN_MINUTES.
	ReReviewCooldownMinutes int

	// DryRun is set by the --dry-run CLI flag (not an env var).
	// When true, reviews are printed to stdout instead of posted on GitHub.
	DryRun bool
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is missing or malformed.
// GitHubReviewerUsername is not set here — call ResolveCurrentUser after
// creating the executor and assign the result to cfg.GitHubReviewerUsername.
func Load() (*Config, error) {
	pollInterval := 15
	if raw := os.Getenv("POLL_INTERVAL_MINUTES"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("POLL_INTERVAL_MINUTES must be a positive integer, got %q", raw)
		}
		pollInterval = v
	}

	model := getenv("CLAUDE_MODEL", "claude-sonnet-4-20250514")
	baseDir := getenv("REPO_BASE_DIR", filepath.Join(os.TempDir(), "watson"))
	sshHost := os.Getenv("GIT_SSH_HOST")

	reviewTTL := 168
	if raw := os.Getenv("REVIEW_TTL_HOURS"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("REVIEW_TTL_HOURS must be a positive integer, got %q", raw)
		}
		reviewTTL = v
	}

	cooldown := 60
	if raw := os.Getenv("RE_REVIEW_COOLDOWN_MINUTES"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("RE_REVIEW_COOLDOWN_MINUTES must be a positive integer, got %q", raw)
		}
		cooldown = v
	}

	return &Config{
		PollIntervalMinutes:     pollInterval,
		ClaudeModel:             model,
		RepoBaseDir:             baseDir,
		GitSSHHost:              sshHost,
		ReviewTTLHours:          reviewTTL,
		ReReviewCooldownMinutes: cooldown,
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
