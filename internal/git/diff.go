package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/meopedevts/watson/internal/github"
)

// DiffResult holds the sanitized diff and supporting context.
type DiffResult struct {
	Diff  string // sanitized unified diff
	Stats string // output of "git diff --stat"
	Note  string // files excluded by sanitization, or truncation warning
}

// GetDiff returns the sanitized diff of the PR branch against the base branch,
// along with the diff stats and a note about any excluded files.
//
// It runs two git commands:
//
//	git -C <cloneDir> diff --stat origin/<baseBranch>..HEAD  (for stats)
//	git -C <cloneDir> diff        origin/<baseBranch>..HEAD  (for full diff)
//
// The base branch is shallow-fetched first so refs/remotes/origin/<base> exists.
func GetDiff(ctx context.Context, exec github.Executor, cloneDir, baseBranch string) (DiffResult, error) {
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", baseBranch, baseBranch)
	if _, err := exec.Run(ctx, "git", "-C", cloneDir, "fetch", "--depth=1", "origin", refspec); err != nil {
		return DiffResult{}, fmt.Errorf("fetch origin/%s in %s: %w", baseBranch, cloneDir, err)
	}

	base := "origin/" + baseBranch

	statsOut, err := exec.Run(ctx, "git", "-C", cloneDir, "diff", "--stat", base+"..HEAD")
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff --stat in %s: %w", cloneDir, err)
	}
	stats := strings.TrimSpace(string(statsOut))

	diffOut, err := exec.Run(ctx, "git", "-C", cloneDir, "diff", base+"..HEAD")
	if err != nil {
		return DiffResult{}, fmt.Errorf("git diff in %s: %w", cloneDir, err)
	}
	rawDiff := strings.TrimRight(string(diffOut), "\n")

	sanitized, note := SanitizeDiff(rawDiff)

	return DiffResult{Diff: sanitized, Stats: stats, Note: note}, nil
}
