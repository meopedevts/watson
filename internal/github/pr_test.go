package github

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
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

func TestFetchPRComments(t *testing.T) {
	raw := `{"comments":[
		{"author":{"login":"alice"},"body":"looks good @watson","createdAt":"2026-03-17T10:00:00Z"},
		{"author":{"login":"bob"},"body":"nice work","createdAt":"2026-03-17T11:00:00Z"}
	]}`
	exec := &MockExecutor{Responses: []MockResponse{{Out: []byte(raw)}}}

	comments, err := FetchPRComments(context.Background(), exec, 42, "org/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].Author.Login != "alice" {
		t.Errorf("expected alice, got %q", comments[0].Author.Login)
	}
	if comments[1].Body != "nice work" {
		t.Errorf("unexpected body: %q", comments[1].Body)
	}
}

func TestFetchPRComments_Error(t *testing.T) {
	exec := &MockExecutor{Responses: []MockResponse{{Err: errors.New("gh: not found")}}}

	_, err := FetchPRComments(context.Background(), exec, 1, "org/repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func mustParseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestFindMentionAfter_ReturnsMention(t *testing.T) {
	cutoff := mustParseTime("2026-03-17T09:00:00Z")
	comments := []Comment{
		{Author: CommentAuthor{Login: "alice"}, Body: "@watson please re-review", CreatedAt: mustParseTime("2026-03-17T10:00:00Z")},
	}

	got := FindMentionAfter(comments, "watson", cutoff)
	if got == nil {
		t.Fatal("expected mention, got nil")
	}
	if got.Author.Login != "alice" {
		t.Errorf("expected alice, got %q", got.Author.Login)
	}
}

func TestFindMentionAfter_NilWhenNone(t *testing.T) {
	cutoff := mustParseTime("2026-03-17T09:00:00Z")
	comments := []Comment{
		{Author: CommentAuthor{Login: "alice"}, Body: "no mention here", CreatedAt: mustParseTime("2026-03-17T10:00:00Z")},
	}

	if got := FindMentionAfter(comments, "watson", cutoff); got != nil {
		t.Errorf("expected nil, got comment from %q", got.Author.Login)
	}
}

func TestFindMentionAfter_IgnoresBeforeCutoff(t *testing.T) {
	cutoff := mustParseTime("2026-03-17T12:00:00Z")
	comments := []Comment{
		{Author: CommentAuthor{Login: "alice"}, Body: "@watson re-review", CreatedAt: mustParseTime("2026-03-17T10:00:00Z")},
	}

	if got := FindMentionAfter(comments, "watson", cutoff); got != nil {
		t.Error("expected nil for comment before cutoff")
	}
}

func TestFindMentionAfter_IgnoresOwnComments(t *testing.T) {
	cutoff := mustParseTime("2026-03-17T09:00:00Z")
	comments := []Comment{
		{Author: CommentAuthor{Login: "watson"}, Body: "@watson this is my own comment", CreatedAt: mustParseTime("2026-03-17T10:00:00Z")},
	}

	if got := FindMentionAfter(comments, "watson", cutoff); got != nil {
		t.Error("expected nil: should not self-trigger")
	}
}

func TestFindMentionAfter_ReturnsFirstMatch(t *testing.T) {
	cutoff := mustParseTime("2026-03-17T09:00:00Z")
	comments := []Comment{
		{Author: CommentAuthor{Login: "alice"}, Body: "@watson first", CreatedAt: mustParseTime("2026-03-17T10:00:00Z")},
		{Author: CommentAuthor{Login: "bob"}, Body: "@watson second", CreatedAt: mustParseTime("2026-03-17T11:00:00Z")},
	}

	got := FindMentionAfter(comments, "watson", cutoff)
	if got == nil {
		t.Fatal("expected mention, got nil")
	}
	if got.Author.Login != "alice" {
		t.Errorf("expected first match from alice, got %q", got.Author.Login)
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
