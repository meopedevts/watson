package git

import (
	"context"
	"errors"
	"testing"

	"github.com/meopedevts/watson/internal/github"
)

// mockExec is a minimal Executor for git package tests.
type mockExec struct {
	responses []struct {
		out []byte
		err error
	}
	idx int
}

func (m *mockExec) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.idx >= len(m.responses) {
		return nil, nil
	}
	r := m.responses[m.idx]
	m.idx++
	return r.out, r.err
}

func (m *mockExec) RunWithStdin(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
	return m.Run(ctx, name, args...)
}

var _ github.Executor = (*mockExec)(nil)

func TestCheckConflicts_DetectsConflictFiles(t *testing.T) {
	mergeOut := "CONFLICT (content): Merge conflict in pkg/handler.go\nCONFLICT (content): Merge conflict in internal/service.go\n"
	exec := &mockExec{responses: []struct {
		out []byte
		err error
	}{
		{out: nil, err: nil},
		{out: []byte(mergeOut), err: errors.New("exit 1")},
	}}

	files, err := CheckConflicts(context.Background(), exec, "/tmp/fake", "master")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 conflict files, got %d: %v", len(files), files)
	}
	if files[0] != "pkg/handler.go" {
		t.Errorf("expected %q, got %q", "pkg/handler.go", files[0])
	}
	if files[1] != "internal/service.go" {
		t.Errorf("expected %q, got %q", "internal/service.go", files[1])
	}
}

func TestCheckConflicts_NoConflicts(t *testing.T) {
	exec := &mockExec{responses: []struct {
		out []byte
		err error
	}{
		{out: nil, err: nil},
		{out: []byte("Merge made by the 'ort' strategy."), err: nil},
	}}

	files, err := CheckConflicts(context.Background(), exec, "/tmp/fake", "master")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected no conflict files, got %v", files)
	}
}

func TestCheckConflicts_FetchError(t *testing.T) {
	exec := &mockExec{responses: []struct {
		out []byte
		err error
	}{
		{out: nil, err: errors.New("network error")},
	}}

	_, err := CheckConflicts(context.Background(), exec, "/tmp/fake", "main")
	if err == nil {
		t.Fatal("expected error from failed fetch")
	}
}

func TestParseConflictFiles(t *testing.T) {
	output := `Auto-merging pkg/handler.go
CONFLICT (content): Merge conflict in pkg/handler.go
Auto-merging internal/service.go
CONFLICT (content): Merge conflict in internal/service.go
Automatic merge failed; fix conflicts and then commit the result.`

	files := parseConflictFiles(output)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	if files[0] != "pkg/handler.go" || files[1] != "internal/service.go" {
		t.Errorf("unexpected files: %v", files)
	}
}
