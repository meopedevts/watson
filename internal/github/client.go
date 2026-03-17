package github

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Executor abstracts shell command execution to make git and gh calls
// injectable and mockable in tests.
type Executor interface {
	// Run executes name with args and returns combined stdout+stderr output.
	Run(ctx context.Context, name string, args ...string) ([]byte, error)

	// RunWithStdin executes name with args, piping stdin into the process.
	// Used to send large prompts to the claude CLI without arg-length limits.
	RunWithStdin(ctx context.Context, stdin string, name string, args ...string) ([]byte, error)
}

// ShellExecutor is the real Executor implementation that delegates to os/exec.
// CombinedOutput is used so that stderr (e.g. git conflict markers) is included
// in the returned bytes, which is required for conflict detection.
type ShellExecutor struct{}

// NewShellExecutor returns a ShellExecutor ready for use.
func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{}
}

// Run executes the command and returns combined stdout+stderr.
func (e *ShellExecutor) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("command %q failed: %w\noutput: %s", name, err, out)
	}
	return out, nil
}

// RunWithStdin executes the command with the given string piped to its stdin.
func (e *ShellExecutor) RunWithStdin(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("command %q failed: %w\noutput: %s", name, err, out)
	}
	return out, nil
}
