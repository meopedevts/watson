package git

import (
	"context"
	"fmt"
	"os"

	"github.com/meopedevts/watson/internal/github"
)

// CloneRepo performs a shallow clone of repoURL at branch into a new
// temporary subdirectory of baseDir. It returns the path to the cloned
// directory. The caller is responsible for cleanup:
//
//	cloneDir, err := git.CloneRepo(...)
//	if err != nil { ... }
//	defer os.RemoveAll(cloneDir)
//
// Equivalent shell command:
//
//	git clone --depth=1 --branch <branch> <repoURL> <tempDir>
func CloneRepo(ctx context.Context, exec github.Executor, repoURL, branch, baseDir string) (string, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return "", fmt.Errorf("create base dir %s: %w", baseDir, err)
	}

	cloneDir, err := os.MkdirTemp(baseDir, "pr-clone-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	_, err = exec.Run(ctx, "git", "clone", "--depth=1", "--branch", branch, repoURL, cloneDir)
	if err != nil {
		os.RemoveAll(cloneDir)
		return "", fmt.Errorf("clone %s@%s: %w", repoURL, branch, err)
	}

	return cloneDir, nil
}
