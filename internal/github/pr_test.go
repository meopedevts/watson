package github

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestListPendingPRs_ReturnsParsedPRs(t *testing.T) {
	raw := `[
		{"number":1,"title":"feat: A","body":"body A","repository":{"nameWithOwner":"org/repo"}},
		{"number":2,"title":"feat: B","body":"","repository":{"nameWithOwner":"org/repo"}}
	]`
	exec := &MockExecutor{Responses: []MockResponse{{Out: []byte(raw)}}}

	prs, err := ListPendingPRs(context.Background(), exec, &sync.Map{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0].Number != 1 || prs[1].Number != 2 {
		t.Errorf("unexpected PR numbers: %v %v", prs[0].Number, prs[1].Number)
	}
}

func TestListPendingPRs_FiltersProcessed(t *testing.T) {
	raw := `[
		{"number":1,"title":"feat: A","body":"","repository":{"nameWithOwner":"org/repo"}},
		{"number":2,"title":"feat: B","body":"","repository":{"nameWithOwner":"org/repo"}},
		{"number":3,"title":"feat: C","body":"","repository":{"nameWithOwner":"org/repo"}}
	]`
	exec := &MockExecutor{Responses: []MockResponse{{Out: []byte(raw)}}}

	processed := &sync.Map{}
	processed.Store(1, struct{}{})
	processed.Store(3, struct{}{})

	prs, err := ListPendingPRs(context.Background(), exec, processed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 1 {
		t.Fatalf("expected 1 PR, got %d", len(prs))
	}
	if prs[0].Number != 2 {
		t.Errorf("expected PR #2, got #%d", prs[0].Number)
	}
}

func TestListPendingPRs_GhError(t *testing.T) {
	exec := &MockExecutor{Responses: []MockResponse{{Err: errors.New("gh: not found")}}}

	_, err := ListPendingPRs(context.Background(), exec, &sync.Map{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPullRequest_RepoURL(t *testing.T) {
	pr := PullRequest{Repository: Repository{NameWithOwner: "org/repo"}}
	if got := pr.RepoURL(); got != "https://github.com/org/repo" {
		t.Errorf("expected %q, got %q", "https://github.com/org/repo", got)
	}
}

func TestPullRequest_CloneURL_HTTPS(t *testing.T) {
	pr := PullRequest{Repository: Repository{NameWithOwner: "org/repo"}}
	if got := pr.CloneURL(""); got != "https://github.com/org/repo" {
		t.Errorf("expected HTTPS URL, got %q", got)
	}
}

func TestPullRequest_CloneURL_SSH(t *testing.T) {
	pr := PullRequest{Repository: Repository{NameWithOwner: "org/repo"}}
	if got := pr.CloneURL("github-snk"); got != "git@github-snk:org/repo.git" {
		t.Errorf("expected SSH URL with alias, got %q", got)
	}
}

func TestFetchPRRefs(t *testing.T) {
	raw := `{"headRefName":"feat/my-branch","baseRefName":"master"}`
	exec := &MockExecutor{Responses: []MockResponse{{Out: []byte(raw)}}}

	refs, err := FetchPRRefs(context.Background(), exec, 42, "org/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if refs.HeadRefName != "feat/my-branch" {
		t.Errorf("HeadRefName: expected %q, got %q", "feat/my-branch", refs.HeadRefName)
	}
	if refs.BaseRefName != "master" {
		t.Errorf("BaseRefName: expected %q, got %q", "master", refs.BaseRefName)
	}
}
