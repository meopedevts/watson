package git

import (
	"context"
	"fmt"
	"strings"

	"github.com/meopedevts/watson/internal/github"
)

// CheckConflicts attempts a dry-run merge of origin/<baseBranch> into the current
// HEAD and returns the list of files with conflicts.
//
// Returns an empty slice when the merge would be clean, a non-empty slice with
// the conflicting file paths when conflicts are detected, or an error when the
// command fails for an unrelated reason.
func CheckConflicts(ctx context.Context, exec github.Executor, cloneDir, baseBranch string) ([]string, error) {
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/origin/%s", baseBranch, baseBranch)
	if _, err := exec.Run(ctx, "git", "-C", cloneDir, "fetch", "--depth=1", "origin", refspec); err != nil {
		return nil, fmt.Errorf("fetch origin/%s in %s: %w", baseBranch, cloneDir, err)
	}

	base := "origin/" + baseBranch
	out, err := exec.Run(ctx, "git", "-C", cloneDir, "merge", "--no-commit", "--no-ff", "--allow-unrelated-histories", base)
	if err != nil {
		files := parseConflictFiles(string(out))
		if len(files) > 0 {
			return files, nil
		}
		return nil, fmt.Errorf("merge check in %s: %w", cloneDir, err)
	}

	return nil, nil
}

// parseConflictFiles extracts file paths from git merge conflict output.
// Git reports conflicts as lines like:
//
//	CONFLICT (content): Merge conflict in path/to/file.go
func parseConflictFiles(output string) []string {
	const marker = "Merge conflict in "
	var files []string
	for _, line := range strings.Split(output, "\n") {
		if i := strings.Index(line, marker); i != -1 {
			if file := strings.TrimSpace(line[i+len(marker):]); file != "" {
				files = append(files, file)
			}
		}
	}
	return files
}
